package crawler

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/gogf/gf/v2/util/grand"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"

	"go-crawler/internal/service"
)

// scrapeProductDetails 获取商品详情数据（并发版本）
func (s *sCrawler) scrapeProductDetails(ctx context.Context, browser *rod.Browser, products []ProductInfo, taskId string) []ProductDetail {
	var details []ProductDetail
	var mu sync.Mutex

	pool := NewWorkerPool(crawlerWorkerConcurrency)

	startTime := time.Now()
	logInfo(ctx, taskId, fmt.Sprintf("🚀 开始并发获取商品详情（并发数=%d，总数=%d）", crawlerWorkerConcurrency, len(products)))

	for i, product := range products {
		// 复制循环变量，避免闭包问题
		index := i
		p := product

		pool.Submit(func() {
			if ctx.Err() != nil {
				return
			}
			detail := s.scrapeSingleProductDetail(ctx, browser, p, taskId)
			if detail != nil {
				mu.Lock()
				details = append(details, *detail)
				mu.Unlock()
				logDebug(ctx, taskId, fmt.Sprintf("✅ [%d/%d] 详情获取完成：%s", index+1, len(products), p.ProductName))
			} else {
				logWarn(ctx, taskId, fmt.Sprintf("⚠️ [%d/%d] 详情获取失败：%s", index+1, len(products), p.ProductName))
			}
		})
	}

	// 等待所有任务完成
	pool.Wait()

	elapsed := time.Since(startTime)
	logInfo(ctx, taskId, fmt.Sprintf("✅ 商品详情获取完成：%d 条，耗时 %.1f 秒（相比串行节省 %.0f 秒）",
		len(details), elapsed.Seconds(), float64(len(products))*2-elapsed.Seconds()))

	return details
}

// scrapeSingleProductDetail 获取单个商品详情（DOM 文本正则提取，与 Python 版本一致）
func (s *sCrawler) scrapeSingleProductDetail(ctx context.Context, browser *rod.Browser, product ProductInfo, taskId string) *ProductDetail {
	// 1. 检查缓存
	cache := service.GetProductDetailCache()
	cachedRecord := cache.GetRecord(product.ProductID)
	if cachedRecord != nil && !cache.IsRecordStale(cachedRecord) && cache.IsRecordComplete(cachedRecord) {
		logInfo(ctx, taskId, fmt.Sprintf("✅ 从缓存获取商品详情：%s", product.ProductName))
		return &ProductDetail{
			ProductID:      product.ProductID,
			ProductName:    product.ProductName,
			FirstShelfTime: cachedRecord.FirstShelfTime,
			MinPrice:       cachedRecord.MinPrice,
			PriceRange:     product.PriceRange,
		}
	}

	// 2. 优先使用罗盘站内详情页。榜单返回的 product_detail_h5_url 常指向
	// haohuo.jinritemai.com 公开商品页，不是罗盘分析详情页。
	detailURL := ""
	if product.ProductID != "" && product.ShopID != "" {
		detailURL = buildDetailURL(product.ProductID, product.ShopID)
	} else {
		detailURL = product.ProductURL
	}
	if detailURL == "" {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 商品 [%s] 无详情 URL，跳过详情获取", product.ProductName))
		return &ProductDetail{ProductName: product.ProductName}
	}

	// 3. 打开新标签页
	var page *rod.Page
	retryErr := s.retryWithBackoff(ctx, func() error {
		var pageErr error
		page, pageErr = browser.Page(proto.TargetCreateTarget{URL: ""})
		return pageErr
	}, 3)
	if retryErr != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 无法打开浏览器页面：%v", retryErr))
		return nil
	}
	defer page.Close()

	// 4. 导航到详情页，使用账号的真实授权和查看配额。
	logDebug(ctx, taskId, fmt.Sprintf("🔗 正在打开商品详情：%s", detailURL))
	if err := s.retryWithBackoff(ctx, func() error {
		return page.Navigate(detailURL)
	}, 3); err != nil {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 打开商品详情失败：%v", err))
		return nil
	}
	page.WaitLoad()

	// 6. 纯 DOM 文本正则提取（核心方式，与 Python 版本一致）
	detail := s.scrapeSingleProductDetailByParsing(ctx, page, product, taskId)

	// 7. 保存到缓存
	if detail != nil && (detail.FirstShelfTime != "" || detail.MinPrice != "") {
		cacheRecord := &service.ProductDetailRecord{
			ProductID:          product.ProductID,
			FirstShelfTime:     detail.FirstShelfTime,
			MinPrice:           detail.MinPrice,
			DetailAccessStatus: detail.DetailAccessStatus,
			ProductName:        product.ProductName,
			ShopName:           product.ShopName,
			DetailURL:          detailURL,
		}
		if _, err := cache.UpsertRecordWithError(cacheRecord); err != nil {
			logWarn(ctx, taskId, fmt.Sprintf("详情缓存落盘失败：%v", err))
		}
	}

	return detail
}

// scrapeSingleProductDetailByParsing 通过 DOM 文本正则提取商品详情（核心方式）
func (s *sCrawler) scrapeSingleProductDetailByParsing(ctx context.Context, page *rod.Page, product ProductInfo, taskId string) *ProductDetail {
	detail := &ProductDetail{
		ProductID:   product.ProductID,
		ProductName: product.ProductName,
	}

	// 随机延迟，模拟真实用户行为
	time.Sleep(time.Duration(grand.Intn(1000)+500) * time.Millisecond)

	// 收集页面文本
	texts := s.collectDetailTexts(ctx, page, taskId)

	// 1. 提取首次上架时间
	detail.FirstShelfTime = s.extractFirstShelfTimeFromTexts(ctx, texts, taskId)

	// 2. 提取最低到手价
	detail.MinPrice = s.extractMinPriceFromTexts(ctx, texts, taskId)

	// 3. 提取运费险信息
	hasInsurance := s.extractShippingInsuranceFromTexts(ctx, texts, taskId)
	detail.HasShippingInsurance = hasInsurance

	// 4. 检查商品状态
	detailAccessStatus, detailAccessNote := s.extractDetailAccessState(ctx, texts, taskId)
	detail.DetailAccessStatus = detailAccessStatus
	detail.DetailAccessNote = detailAccessNote

	if detail.FirstShelfTime == "" && detail.MinPrice == "" {
		logWarn(ctx, taskId, fmt.Sprintf("⚠️ 商品 [%s] 详情页未提取到数据，可能需要更新选择器", product.ProductName))
	}

	return detail
}

// collectDetailTexts 收集页面文本（借鉴 Python 爬虫的 _collect_detail_texts）
func (s *sCrawler) collectDetailTexts(ctx context.Context, page *rod.Page, taskId string) []string {
	// 定义关键词，用于判断页面是否加载完成
	keywords := []string{"首次上架", "最低到手价", "服务保障", "运费险"}

	// 重试机制，最多尝试 4 次
	for attempt := 0; attempt < 4; attempt++ {
		var texts []string

		// 尝试从多个元素获取文本
		selectors := []string{"body", "main", "#root"}
		for _, selector := range selectors {
			if elem, err := page.Element(selector); err == nil {
				if text, err := elem.Text(); err == nil && text != "" {
					texts = append(texts, text)
				}
			}
		}

		// 如果没有获取到文本，尝试获取页面内容
		if len(texts) == 0 {
			if content, err := page.HTML(); err == nil {
				texts = append(texts, content)
			}
		}

		// 检查是否包含关键词
		for _, text := range texts {
			for _, keyword := range keywords {
				if strings.Contains(text, keyword) {
					logDebug(ctx, taskId, fmt.Sprintf("✅ 页面文本包含关键词：%s", keyword))
					return texts
				}
			}
		}

		// 等待后重试
		if attempt < 3 {
			time.Sleep(800 * time.Millisecond)
		}
	}

	// 返回最后一次收集的文本
	var texts []string
	selectors := []string{"body", "main", "#root"}
	for _, selector := range selectors {
		if elem, err := page.Element(selector); err == nil {
			if text, err := elem.Text(); err == nil && text != "" {
				texts = append(texts, text)
				break
			}
		}
	}

	if len(texts) == 0 {
		if content, err := page.HTML(); err == nil {
			texts = append(texts, content)
		}
	}

	return texts
}

// extractFirstShelfTimeFromTexts 从文本中提取首次上架时间
func (s *sCrawler) extractFirstShelfTimeFromTexts(ctx context.Context, texts []string, taskId string) string {
	// 定义多种日期格式的正则表达式
	patterns := []string{
		`首次上架\s*[:：]?\s*(\d{4}\s*年\s*\d{1,2}\s*月\s*\d{1,2}\s*日?)`,
		`首次上架\s*[:：]?\s*(\d{4}[./-]\d{1,2}[./-]\d{1,2})`,
		`上架时间\s*[:：]?\s*(\d{4}\s*年\s*\d{1,2}\s*月\s*\d{1,2}\s*日?)`,
		`上架时间\s*[:：]?\s*(\d{4}[./-]\d{1,2}[./-]\d{1,2})`,
	}

	for _, text := range texts {
		for _, pattern := range patterns {
			if matches := regexp.MustCompile(pattern).FindStringSubmatch(text); len(matches) > 1 {
				dateStr := strings.TrimSpace(matches[1])
				// 标准化日期格式
				dateStr = s.normalizeDateString(dateStr)
				logDebug(ctx, taskId, fmt.Sprintf("📅 提取到上架时间：%s", dateStr))
				return dateStr
			}
		}
	}

	return ""
}

// extractMinPriceFromTexts 从文本中提取最低到手价
func (s *sCrawler) extractMinPriceFromTexts(ctx context.Context, texts []string, taskId string) string {
	// 定义价格的正则表达式
	patterns := []string{
		`最低到手价\s*[:：]?\s*[¥￥]?\s*([\d,\s]+(?:\s*\.\s*\d+)?)`,
		`到手价\s*[:：]?\s*[¥￥]?\s*([\d,\s]+(?:\s*\.\s*\d+)?)`,
		`最低价\s*[:：]?\s*[¥￥]?\s*([\d,\s]+(?:\s*\.\s*\d+)?)`,
	}

	for _, text := range texts {
		for _, pattern := range patterns {
			if matches := regexp.MustCompile(pattern).FindStringSubmatch(text); len(matches) > 1 {
				priceStr := strings.TrimSpace(matches[1])
				// 清理价格字符串
				priceStr = strings.ReplaceAll(priceStr, " ", "")
				priceStr = strings.ReplaceAll(priceStr, ",", "")
				logDebug(ctx, taskId, fmt.Sprintf("💰 提取到最低到手价：%s", priceStr))
				return priceStr
			}
		}
	}

	return ""
}

// extractShippingInsuranceFromTexts 从文本中提取运费险信息
func (s *sCrawler) extractShippingInsuranceFromTexts(ctx context.Context, texts []string, taskId string) *bool {
	shippingInsuranceMarkers := []string{"配置运费险", "赠送运费险", "运费险"}
	noShippingInsuranceMarkers := []string{"无运费险", "未配置运费险"}
	sawServiceGuarantee := false

	for _, text := range texts {
		for _, marker := range shippingInsuranceMarkers {
			if strings.Contains(text, marker) {
				trueValue := true
				logDebug(ctx, taskId, "📦 提取到运费险：是")
				return &trueValue
			}
		}
		for _, marker := range noShippingInsuranceMarkers {
			if strings.Contains(text, marker) {
				falseValue := false
				logDebug(ctx, taskId, "📦 提取到运费险：否")
				return &falseValue
			}
		}
		if strings.Contains(text, "服务保障") {
			sawServiceGuarantee = true
		}
	}

	if sawServiceGuarantee {
		return nil
	}

	return nil
}

// extractDetailAccessState 提取商品详情访问状态
func (s *sCrawler) extractDetailAccessState(ctx context.Context, texts []string, taskId string) (string, string) {
	removedMarkers := []string{"目标商品已下架"}
	categoryMismatchMarkers := []string{
		"您经营的类目与对方不匹配",
		"经营的类目与对方不匹配",
		"类目与对方不匹配",
	}

	for _, text := range texts {
		for _, marker := range removedMarkers {
			if strings.Contains(text, marker) {
				logDebug(ctx, taskId, "⚠️ 商品已下架")
				return "removed", "目标商品已下架"
			}
		}
		for _, marker := range categoryMismatchMarkers {
			if strings.Contains(text, marker) {
				logDebug(ctx, taskId, "⚠️ 类目不匹配")
				return "category_mismatch", "经营类目不匹配"
			}
		}
	}

	return "available", ""
}

// normalizeDateString 标准化日期字符串
func (s *sCrawler) normalizeDateString(dateStr string) string {
	// 替换各种日期分隔符
	dateStr = strings.ReplaceAll(dateStr, "年", "-")
	dateStr = strings.ReplaceAll(dateStr, "月", "-")
	dateStr = strings.ReplaceAll(dateStr, "日", "")
	dateStr = strings.ReplaceAll(dateStr, ".", "-")
	dateStr = strings.ReplaceAll(dateStr, "/", "-")

	// 清理空格
	dateStr = strings.ReplaceAll(dateStr, " ", "")

	return dateStr
}

// processResponseBody 处理响应体的编码和压缩
func (s *sCrawler) processResponseBody(body []byte, headers map[string][]string) (string, error) {
	// 检查是否是gzip压缩
	if encoding, ok := headers["Content-Encoding"]; ok {
		for _, enc := range encoding {
			if strings.Contains(enc, "gzip") {
				// 解压缩gzip数据
				gzipReader, err := gzip.NewReader(bytes.NewReader(body))
				if err != nil {
					return "", err
				}
				defer gzipReader.Close()

				uncompressed, err := io.ReadAll(gzipReader)
				if err != nil {
					return "", err
				}
				body = uncompressed
				break
			}
		}
	}

	// 检查是否是deflate压缩
	if encoding, ok := headers["Content-Encoding"]; ok {
		for _, enc := range encoding {
			if strings.Contains(enc, "deflate") {
				// 解压缩deflate数据
				var buf bytes.Buffer
				deflater := flate.NewReader(bytes.NewReader(body))
				if _, err := io.Copy(&buf, deflater); err != nil {
					return "", err
				}
				deflater.Close()
				body = buf.Bytes()
				break
			}
		}
	}

	// 尝试使用不同的编码解码
	// 先尝试 UTF-8
	if utf8Content := string(body); len(utf8Content) > 0 {
		return utf8Content, nil
	}

	// 尝试 GBK
	if reader := transform.NewReader(bytes.NewReader(body), simplifiedchinese.GBK.NewDecoder()); reader != nil {
		if content, err := io.ReadAll(reader); err == nil && len(content) > 0 {
			return string(content), nil
		}
	}

	// 尝试 GB2312
	if reader := transform.NewReader(bytes.NewReader(body), simplifiedchinese.HZGB2312.NewDecoder()); reader != nil {
		if content, err := io.ReadAll(reader); err == nil && len(content) > 0 {
			return string(content), nil
		}
	}

	// 尝试 Big5
	if reader := transform.NewReader(bytes.NewReader(body), traditionalchinese.Big5.NewDecoder()); reader != nil {
		if content, err := io.ReadAll(reader); err == nil && len(content) > 0 {
			return string(content), nil
		}
	}

	// 直接尝试转换为字符串
	return string(body), nil
}

// parseProductDetailResponse 解析商品详情 API 响应 JSON
func parseProductDetailResponse(body string, product ProductInfo) *ProductDetail {
	detail := &ProductDetail{
		ProductName: product.ProductName,
	}

	// 打印响应体前1000字符，了解API响应结构
	if len(body) > 1000 {
		fmt.Printf("[DEBUG] 商品详情API响应前1000字符: %s\n", body[:1000])
	} else {
		fmt.Printf("[DEBUG] 商品详情API响应: %s\n", body)
	}

	// 尝试多种API响应结构解析
	// 结构1: {"data":{"first_shelf_time":"...", "min_price":"..."}}
	type response1 struct {
		Data struct {
			FirstShelfTime       string `json:"first_shelf_time"`
			MinPrice             string `json:"min_price"`
			HasShippingInsurance *bool  `json:"has_shipping_insurance"`
		} `json:"data"`
	}

	var resp1 response1
	if err := json.Unmarshal([]byte(body), &resp1); err == nil {
		if resp1.Data.FirstShelfTime != "" {
			detail.FirstShelfTime = resp1.Data.FirstShelfTime
		}
		if resp1.Data.MinPrice != "" {
			detail.MinPrice = resp1.Data.MinPrice
		}
		fmt.Printf("[DEBUG] 解析商品详情API响应成功：结构1\n")
		return detail
	}

	// 结构2: {"data":{"product_info":{"first_shelf_time":"...", "min_price":"..."}}}
	type response2 struct {
		Data struct {
			ProductInfo struct {
				FirstShelfTime       string `json:"first_shelf_time"`
				MinPrice             string `json:"min_price"`
				HasShippingInsurance *bool  `json:"has_shipping_insurance"`
			} `json:"product_info"`
		} `json:"data"`
	}

	var resp2 response2
	if err := json.Unmarshal([]byte(body), &resp2); err == nil {
		if resp2.Data.ProductInfo.FirstShelfTime != "" {
			detail.FirstShelfTime = resp2.Data.ProductInfo.FirstShelfTime
		}
		if resp2.Data.ProductInfo.MinPrice != "" {
			detail.MinPrice = resp2.Data.ProductInfo.MinPrice
		}
		fmt.Printf("[DEBUG] 解析商品详情API响应成功：结构2\n")
		return detail
	}

	// 结构3: {"result":{"first_shelf_time":"...", "min_price":"..."}}
	type response3 struct {
		Result struct {
			FirstShelfTime       string `json:"first_shelf_time"`
			MinPrice             string `json:"min_price"`
			HasShippingInsurance *bool  `json:"has_shipping_insurance"`
		} `json:"result"`
	}

	var resp3 response3
	if err := json.Unmarshal([]byte(body), &resp3); err == nil {
		if resp3.Result.FirstShelfTime != "" {
			detail.FirstShelfTime = resp3.Result.FirstShelfTime
		}
		if resp3.Result.MinPrice != "" {
			detail.MinPrice = resp3.Result.MinPrice
		}
		fmt.Printf("[DEBUG] 解析商品详情API响应成功：结构3\n")
		return detail
	}

	// 结构4: {"data":{"item_info":{"first_shelf_time":"...", "min_price":"..."}}}
	type response4 struct {
		Data struct {
			ItemInfo struct {
				FirstShelfTime       string `json:"first_shelf_time"`
				MinPrice             string `json:"min_price"`
				HasShippingInsurance *bool  `json:"has_shipping_insurance"`
			} `json:"item_info"`
		} `json:"data"`
	}

	var resp4 response4
	if err := json.Unmarshal([]byte(body), &resp4); err == nil {
		if resp4.Data.ItemInfo.FirstShelfTime != "" {
			detail.FirstShelfTime = resp4.Data.ItemInfo.FirstShelfTime
		}
		if resp4.Data.ItemInfo.MinPrice != "" {
			detail.MinPrice = resp4.Data.ItemInfo.MinPrice
		}
		fmt.Printf("[DEBUG] 解析商品详情API响应成功：结构4\n")
		return detail
	}

	// 结构5: 直接在根级别
	type response5 struct {
		FirstShelfTime       string `json:"first_shelf_time"`
		MinPrice             string `json:"min_price"`
		HasShippingInsurance *bool  `json:"has_shipping_insurance"`
	}

	var resp5 response5
	if err := json.Unmarshal([]byte(body), &resp5); err == nil {
		if resp5.FirstShelfTime != "" {
			detail.FirstShelfTime = resp5.FirstShelfTime
		}
		if resp5.MinPrice != "" {
			detail.MinPrice = resp5.MinPrice
		}
		fmt.Printf("[DEBUG] 解析商品详情API响应成功：结构5\n")
		return detail
	}

	// 尝试使用正则表达式提取关键信息
	detail = extractDetailByRegex(body, detail)
	if detail.FirstShelfTime != "" || detail.MinPrice != "" {
		fmt.Printf("[DEBUG] 解析商品详情API响应成功：正则表达式\n")
		return detail
	}

	// 解析失败，返回空详情
	fmt.Printf("[DEBUG] 解析商品详情API响应失败：所有结构都不匹配\n")
	return detail
}

// extractDetailByRegex 使用正则表达式从响应体中提取商品详情
func extractDetailByRegex(body string, detail *ProductDetail) *ProductDetail {
	// 提取首次上架时间
	datePatterns := []*regexp.Regexp{
		regexp.MustCompile(`"first_shelf_time"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`first_shelf_time\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`上架时间\s*:\s*"([^"]+)"`),
	}

	for _, pattern := range datePatterns {
		matches := pattern.FindStringSubmatch(body)
		if len(matches) > 1 {
			detail.FirstShelfTime = matches[1]
			break
		}
	}

	// 提取最低到手价
	pricePatterns := []*regexp.Regexp{
		regexp.MustCompile(`"min_price"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`min_price\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`最低价格\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`"price"\s*:\s*"([^"]+)"`),
		regexp.MustCompile(`price\s*:\s*"([^"]+)"`),
	}

	for _, pattern := range pricePatterns {
		matches := pattern.FindStringSubmatch(body)
		if len(matches) > 1 {
			detail.MinPrice = matches[1]
			break
		}
	}

	return detail
}
