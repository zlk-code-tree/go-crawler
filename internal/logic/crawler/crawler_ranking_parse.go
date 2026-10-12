package crawler

import (
	"encoding/json"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// parseProductRankResponse 解析商品榜单 API 响应 JSON
func parseProductRankResponse(body string) []ProductInfo {
	// 使用简单的 JSON 解析，避免引入额外依赖
	// 结构：data.data_result[i].product_info.*
	type shopInfo struct {
		ShopID        string `json:"shop_id"`
		ShopName      string `json:"shop_name"`
		ShopLogoURL   string `json:"shop_logo_url"`
		LogoURL       string `json:"logo_url"`
		ShopAvatarURL string `json:"shop_avatar_url"`
		AvatarURL     string `json:"avatar_url"`
		ShopIconURL   string `json:"shop_icon_url"`
		ShopQRCodeURL string `json:"shop_qr_code_url"`
	}
	type productInfo struct {
		Rank             int        `json:"rank"`
		ID               string     `json:"id"`
		Name             string     `json:"name"`
		PriceBin         string     `json:"price_bin"`
		ImageURL         string     `json:"image_url"`
		ProductImageURL  string     `json:"product_image_url"`
		ProductImgURL    string     `json:"product_img_url"`
		MainImageURL     string     `json:"main_image_url"`
		CoverURL         string     `json:"cover_url"`
		PicURL           string     `json:"pic_url"`
		ImgURL           string     `json:"img_url"`
		QRCodeURL        string     `json:"qr_code_url"`
		ProductQRCodeURL string     `json:"product_qr_code_url"`
		ProductDetailURL string     `json:"product_detail_h5_url"`
		NewlyOnRanking   bool       `json:"newly_on_ranking"`
		ShopList         []shopInfo `json:"shop_list"`
	}
	type valueItem struct {
		Value interface{} `json:"value"`
		Unit  string      `json:"unit"`
	}
	type valueRange struct {
		ValueRange []valueItem `json:"value_range"`
	}
	type dataItem struct {
		ProductInfo productInfo `json:"product_info"`
		PayAmt      valueRange  `json:"pay_amt"`
		NewPayAmt   valueRange  `json:"new_pay_amt"`
		PayComboCnt valueRange  `json:"pay_combo_cnt"`
		ClickCnt    valueRange  `json:"product_click_cnt"`
		VideoCnt    valueRange  `json:"video_cnt"`
	}
	type pageResult struct {
		Total    int `json:"total"`
		PageNo   int `json:"page_no"`
		PageSize int `json:"page_size"`
	}
	type dataWrapper struct {
		DataResult []dataItem `json:"data_result"`
		PageResult pageResult `json:"page_result"`
	}
	type apiResponse struct {
		Data dataWrapper `json:"data"`
	}
	type rawAPIResponse struct {
		Data struct {
			DataResult []map[string]any `json:"data_result"`
		} `json:"data"`
	}

	var resp apiResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil
	}
	var rawResp rawAPIResponse
	_ = json.Unmarshal([]byte(body), &rawResp)

	formatNumberWithChineseUnit := func(value float64, unit string) string {
		unit = strings.TrimSpace(unit)
		if unit == "number" || unit == "cnt" || unit == "count" || unit == "item" ||
			unit == "fen" || unit == "cent" || unit == "cents" || unit == "yuan" {
			unit = ""
		}
		if unit == "%" {
			return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 2, 64), "0"), ".") + "%"
		}
		if math.Abs(value) >= 100000000 {
			return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value/100000000, 'f', 2, 64), "0"), ".") + "亿"
		}
		if math.Abs(value) >= 10000 {
			return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value/10000, 'f', 2, 64), "0"), ".") + "万"
		}
		text := strings.TrimRight(strings.TrimRight(strconv.FormatFloat(value, 'f', 2, 64), "0"), ".")
		if unit != "" {
			return text + unit
		}
		return text
	}
	formatMetricItemWithScale := func(item valueItem, scale float64) string {
		if scale == 0 {
			scale = 1
		}
		switch v := item.Value.(type) {
		case string:
			text := strings.TrimSpace(v)
			if number, err := strconv.ParseFloat(strings.ReplaceAll(text, ",", ""), 64); err == nil {
				return formatNumberWithChineseUnit(number*scale, item.Unit)
			}
			if item.Unit != "" && !strings.Contains(text, item.Unit) {
				return text + item.Unit
			}
			return text
		case float64:
			return formatNumberWithChineseUnit(v*scale, item.Unit)
		case int:
			return formatNumberWithChineseUnit(float64(v)*scale, item.Unit)
		case int64:
			return formatNumberWithChineseUnit(float64(v)*scale, item.Unit)
		case json.Number:
			if number, err := v.Float64(); err == nil {
				return formatNumberWithChineseUnit(number*scale, item.Unit)
			}
			return v.String()
		}
		return ""
	}
	formatRangeWithScale := func(vr valueRange, scale float64) string {
		items := vr.ValueRange
		if len(items) == 0 {
			return ""
		}
		first := items[0]
		last := items[len(items)-1]
		firstValue := formatMetricItemWithScale(first, scale)
		lastValue := formatMetricItemWithScale(last, scale)

		if firstValue == "" {
			return ""
		}
		if firstValue == lastValue {
			return firstValue
		}
		return firstValue + "-" + lastValue
	}
	formatRange := func(vr valueRange) string {
		return formatRangeWithScale(vr, 1)
	}

	var products []ProductInfo
	for i, item := range resp.Data.DataResult {
		info := item.ProductInfo
		var rawItem map[string]any
		if i < len(rawResp.Data.DataResult) {
			rawItem = rawResp.Data.DataResult[i]
		}
		if info.ID == "" || info.Name == "" {
			continue
		}
		shopName := ""
		shopID := ""
		shopLogoURL := ""
		shopQRCodeURL := ""
		if len(info.ShopList) > 0 {
			shopName = info.ShopList[0].ShopName
			shopID = info.ShopList[0].ShopID
			shopLogoURL = firstNonEmptyString(
				info.ShopList[0].ShopLogoURL,
				info.ShopList[0].LogoURL,
				info.ShopList[0].ShopAvatarURL,
				info.ShopList[0].AvatarURL,
				info.ShopList[0].ShopIconURL,
			)
			shopQRCodeURL = info.ShopList[0].ShopQRCodeURL
		}
		rawProductInfo := any(nil)
		rawShopInfo := any(nil)
		if rawItem != nil {
			rawProductInfo = rawItem["product_info"]
			if productInfoMap, ok := rawProductInfo.(map[string]any); ok {
				if shops, ok := productInfoMap["shop_list"].([]any); ok && len(shops) > 0 {
					rawShopInfo = shops[0]
				}
			}
		}
		firstList := 0
		if info.NewlyOnRanking {
			firstList = 1
		}
		// 优先使用 pay_amt，回退到 new_pay_amt
		payAmt := item.PayAmt
		if len(payAmt.ValueRange) == 0 {
			payAmt = item.NewPayAmt
		}
		productImageURL := firstNonEmptyString(
			info.ImageURL,
			info.ProductImageURL,
			info.ProductImgURL,
			info.MainImageURL,
			info.CoverURL,
			info.PicURL,
			info.ImgURL,
			firstRecursiveString(rawProductInfo,
				"image_url",
				"product_image_url",
				"product_img_url",
				"main_image_url",
				"cover_url",
				"pic_url",
				"img_url",
				"image",
				"product_image",
				"head_image_url",
			),
			firstRecursiveString(rawItem,
				"product_image_url",
				"product_img_url",
				"main_image_url",
				"product_cover_url",
				"product_pic_url",
			),
		)
		productQRCodeURL := firstNonEmptyString(
			info.ProductQRCodeURL,
			info.QRCodeURL,
			firstRecursiveString(rawProductInfo, "product_qr_code_url", "product_qrcode_url", "product_qr_url", "qr_code_url", "qrcode_url"),
			firstRecursiveString(rawItem, "product_qr_code_url", "product_qrcode_url", "product_qr_url"),
		)
		shopLogoURL = firstNonEmptyString(
			shopLogoURL,
			firstRecursiveString(rawShopInfo,
				"shop_logo_url",
				"shop_logo",
				"logo_url",
				"shop_avatar",
				"shop_avatar_url",
				"shop_icon",
				"shop_icon_url",
				"avatar_url",
				"avatar",
				"author_avatar",
				"account_avatar",
				"head_image_url",
			),
			firstRecursiveString(rawItem,
				"shop_logo_url",
				"shop_avatar_url",
				"shop_icon_url",
				"shop_head_img",
			),
		)
		shopQRCodeURL = firstNonEmptyString(
			shopQRCodeURL,
			firstRecursiveString(rawShopInfo, "shop_qr_code_url", "shop_qrcode_url", "shop_qr_url", "qr_code_url", "qrcode_url"),
			firstRecursiveString(rawItem, "shop_qr_code_url", "shop_qrcode_url", "shop_qr_url"),
		)
		previousRank, rankDelta, rankTrendSource := extractOfficialRankTrend(rawItem, info.Rank)
		product := ProductInfo{
			Rank:          info.Rank,
			ProductID:     info.ID,
			ProductName:   info.Name,
			ImageURL:      productImageURL,
			ShopID:        shopID,
			ShopName:      shopName,
			PaymentAmount: formatRangeWithScale(payAmt, 0.01),
			ClickCount:    formatRange(item.ClickCnt),
			ClickConvertRate: firstMetricString(rawItem,
				"click_convert_rate",
				"click_conversion",
				"click_conversion_rate",
				"click_pay_ratio",
				"click_pay_rate",
				"pay_click_ratio",
				"pay_click_rate",
				"product_click_pay_ratio",
				"product_click_pay_rate",
				"product_click_convert_rate",
				"product_click_conversion_rate",
				"product_click_pay_combo_rate",
				"pay_combo_click_ratio",
			),
			SalesVolume:      formatRange(item.PayComboCnt),
			DeliveryVideoNum: formatRange(item.VideoCnt),
			ProductQRCodeURL: productQRCodeURL,
			ShopLogoURL:      shopLogoURL,
			ShopQRCodeURL:    shopQRCodeURL,
			PreviousRank:     previousRank,
			RankDelta:        rankDelta,
			RankTrendSource:  rankTrendSource,
			ProductURL:       info.ProductDetailURL,
			FirstList:        firstList,
			PriceRange:       info.PriceBin,
		}
		products = append(products, product)
	}
	return products
}

func extractOfficialRankTrend(rawItem map[string]any, currentRank int) (*int, *int, string) {
	if len(rawItem) == 0 || currentRank <= 0 {
		return nil, nil, ""
	}
	previousRank := firstIntByRecursiveKeys(rawItem,
		"previous_rank", "previousRank", "prev_rank", "prevRank",
		"pre_rank", "preRank", "last_rank", "lastRank",
		"last_period_rank", "lastPeriodRank", "prior_rank", "priorRank",
		"before_rank", "beforeRank",
	)
	rankDelta := firstIntByRecursiveKeys(rawItem,
		"rank_delta", "rankDelta", "rank_diff", "rankDiff",
		"rank_change", "rankChange", "rank_change_num", "rankChangeNum",
		"rank_change_value", "rankChangeValue", "rank_trend_num", "rankTrendNum",
		"rank_trend_value", "rankTrendValue", "rank_offset", "rankOffset",
		"ranking_change", "rankingChange", "change_rank", "changeRank",
		"flow_index", "flowIndex",
	)
	trendText := firstStringByRecursiveKeys(rawItem,
		"rank_trend", "rankTrend", "rank_direction", "rankDirection",
		"rank_change_type", "rankChangeType", "rank_status", "rankStatus",
		"trend", "trend_text", "trendText", "change_text", "changeText",
		"flow_index", "flowIndex",
	)
	textDelta, textOK := parseRankTrendText(trendText)
	if textOK {
		rankDelta = &textDelta
	} else if rankDelta != nil {
		switch rankTrendDirection(trendText) {
		case "up":
			value := absInt(*rankDelta)
			rankDelta = &value
		case "down":
			value := -absInt(*rankDelta)
			rankDelta = &value
		}
	}
	if previousRank != nil {
		if rankDelta == nil {
			delta := *previousRank - currentRank
			rankDelta = &delta
		}
		return previousRank, rankDelta, "official"
	}
	if rankDelta != nil {
		previous := currentRank + *rankDelta
		if previous > 0 {
			previousRank = &previous
		}
		return previousRank, rankDelta, "official"
	}
	return nil, nil, ""
}

func firstIntByRecursiveKeys(value any, keys ...string) *int {
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[strings.ToLower(key)] = struct{}{}
	}
	return findIntByKey(value, keySet, 0)
}

func findIntByKey(value any, keySet map[string]struct{}, depth int) *int {
	if depth > 5 {
		return nil
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if _, ok := keySet[strings.ToLower(key)]; ok {
				if number, ok := intFromAny(child); ok {
					return &number
				}
				if textDelta, ok := parseRankTrendText(stringFromJSONValue(child)); ok {
					return &textDelta
				}
			}
		}
		for _, child := range typed {
			if found := findIntByKey(child, keySet, depth+1); found != nil {
				return found
			}
		}
	case []any:
		for _, child := range typed {
			if found := findIntByKey(child, keySet, depth+1); found != nil {
				return found
			}
		}
	}
	return nil
}

func firstStringByRecursiveKeys(value any, keys ...string) string {
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[strings.ToLower(key)] = struct{}{}
	}
	return findStringByKey(value, keySet, 0)
}

func findStringByKey(value any, keySet map[string]struct{}, depth int) string {
	if depth > 5 {
		return ""
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if _, ok := keySet[strings.ToLower(key)]; ok {
				if text := strings.TrimSpace(stringFromJSONValue(child)); text != "" {
					return text
				}
			}
		}
		for _, child := range typed {
			if found := findStringByKey(child, keySet, depth+1); found != "" {
				return found
			}
		}
	case []any:
		for _, child := range typed {
			if found := findStringByKey(child, keySet, depth+1); found != "" {
				return found
			}
		}
	}
	return ""
}

func intFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	case json.Number:
		number, err := typed.Int64()
		return int(number), err == nil
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return 0, false
		}
		if delta, ok := parseRankTrendText(text); ok {
			return delta, true
		}
		text = strings.TrimPrefix(text, "+")
		number, err := strconv.Atoi(text)
		return number, err == nil
	default:
		return 0, false
	}
}

func parseRankTrendText(text string) (int, bool) {
	text = strings.TrimSpace(text)
	if text == "" || text == "-" || text == "持平" {
		return 0, text == "持平"
	}
	pattern := regexp.MustCompile(`([↑↓]|上升|上涨|提升|升|下降|下跌|降低|降)\s*([+-]?\d+)`)
	match := pattern.FindStringSubmatch(text)
	if len(match) < 3 {
		return 0, false
	}
	number, err := strconv.Atoi(strings.TrimPrefix(match[2], "+"))
	if err != nil {
		return 0, false
	}
	switch match[1] {
	case "↓", "下降", "下跌", "降低", "降":
		return -absInt(number), true
	default:
		return absInt(number), true
	}
}

func rankTrendDirection(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	switch {
	case strings.Contains(text, "↓"), strings.Contains(text, "下降"), strings.Contains(text, "下跌"),
		strings.Contains(text, "降低"), strings.Contains(text, "down"), strings.Contains(text, "fall"):
		return "down"
	case strings.Contains(text, "↑"), strings.Contains(text, "上升"), strings.Contains(text, "上涨"),
		strings.Contains(text, "提升"), strings.Contains(text, "up"), strings.Contains(text, "rise"):
		return "up"
	default:
		return ""
	}
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
