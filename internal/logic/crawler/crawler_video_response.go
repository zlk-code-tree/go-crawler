package crawler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// FlowMoreResponse flow_more API 完整响应
type FlowMoreResponse struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Msg     string       `json:"msg"`
	Data    FlowMoreData `json:"data"`
}

// FlowMoreData flow_more 响应 data 层
type FlowMoreData struct {
	VideoList []FlowVideoItem `json:"video_list"`
	Total     int             `json:"total"`
	HasMore   bool            `json:"has_more"`
}

// FlowVideoItem flow_more 视频条目（深层嵌套结构）
type FlowVideoItem struct {
	BaseInfo struct {
		VideoBaseInfo struct {
			VideoID     string `json:"video_id"`
			VideoTitle  string `json:"video_title"`
			VideoImage  string `json:"video_image"`
			PublishTime string `json:"publish_time"`
			AuthorInfo  struct {
				AuthorNickName string `json:"author_nick_name"`
			} `json:"author_info"`
		} `json:"video_base_info"`
	} `json:"base_info"`
	Metrics FlowMetrics `json:"metrics"`
}

func convertFlowVideoItems(items []FlowVideoItem, product ProductInfo) []VideoAnalysis {
	videos := make([]VideoAnalysis, 0, len(items))
	for _, item := range items {
		base := item.BaseInfo.VideoBaseInfo
		if base.VideoID == "" {
			continue
		}
		videos = append(videos, VideoAnalysis{
			ProductID: product.ProductID, ProductName: product.ProductName,
			VideoId: base.VideoID, VideoDesc: base.VideoTitle,
			AuthorName: base.AuthorInfo.AuthorNickName, CoverURL: base.VideoImage,
			PublishTime: base.PublishTime, SourceURL: fmt.Sprintf("https://www.douyin.com/video/%s", base.VideoID),
			PaymentAmount: item.Metrics.PayAmt.FormatRange(), ExposeUserCount: item.Metrics.ShowUV.FormatRange(),
			ClickUserCount: item.Metrics.DhProductClickUV.FormatRange(), ConvertUserCount: item.Metrics.PayUcnt.FormatRange(),
			ClickRate: item.Metrics.ClickRatio.FormatRange(), ClickConvertRate: item.Metrics.ClickConversion.FormatRange(),
			ExposeConvertRate: item.Metrics.ShowConversion.FormatRange(), OrderCount: item.Metrics.PayCnt.FormatRange(),
			SalesVolume: item.Metrics.PayComboCnt.FormatRange(),
		})
	}
	return videos
}

// FlowMetrics 指标集合
type FlowMetrics struct {
	PayAmt           MetricField `json:"pay_amt"`
	DhProductClickUV MetricField `json:"dh_product_click_uv"`
	PayComboCnt      MetricField `json:"pay_combo_cnt"`
	ShowUV           MetricField `json:"show_uv"`
	PayUcnt          MetricField `json:"pay_ucnt"`
	ClickRatio       MetricField `json:"click_ratio"`
	ClickConversion  MetricField `json:"click_conversion"`
	ShowConversion   MetricField `json:"show_conversion"`
	PayCnt           MetricField `json:"pay_cnt"`
}

// MetricField 指标字段（包含 value 范围和单位）
type MetricField struct {
	Value      MetricValue `json:"value"`
	ExtraValue string      `json:"extra_value"`
	Unit       string      `json:"unit"`
}

// MetricValue 指标值范围
type MetricValue struct {
	Lower float64 `json:"lower"`
	Upper float64 `json:"upper"`
}

// FormatRange 格式化指标范围（处理单位转换：分→元、小数→百分比）
func (m MetricField) FormatRange() string {
	lower, upper := m.Value.Lower, m.Value.Upper
	switch m.Unit {
	case "fen": // 分 → 元
		lower /= 100
		upper /= 100
		return fmt.Sprintf("%.0f-%.0f", lower, upper)
	case "ratio": // 小数 → 百分比
		return fmt.Sprintf("%.2f%%-%.2f%%", lower*100, upper*100)
	case "wan": // 万
		return fmt.Sprintf("%.1f万-%.1f万", lower/10000, upper/10000)
	default:
		if lower == 0 && upper == 0 {
			return ""
		}
		return fmt.Sprintf("%.0f-%.0f", lower, upper)
	}
}

// FormatFloor 取范围下限值（用于 total_gmv 等排序场景）
func (m MetricField) FormatFloor() string {
	switch m.Unit {
	case "fen":
		return fmt.Sprintf("%.0f", m.Value.Lower/100)
	default:
		return fmt.Sprintf("%.0f", m.Value.Lower)
	}
}

// parseVideoAnalysisResponse 解析视频分析 API 响应 JSON
func parseVideoAnalysisResponse(body string, product ProductInfo) []VideoAnalysis {
	var nested FlowMoreResponse
	if err := json.Unmarshal([]byte(body), &nested); err == nil {
		if videos := convertFlowVideoItems(nested.Data.VideoList, product); len(videos) > 0 {
			return videos
		}
	}

	// 兼容早期扁平结构，便于在平台灰度切换期间继续工作。
	type videoItem struct {
		VideoId           string `json:"video_id"`
		VideoDesc         string `json:"video_desc"`
		PaymentAmount     string `json:"payment_amount"`
		ExposeUserCount   string `json:"expose_user_count"`
		ClickUserCount    string `json:"click_user_count"`
		ConvertUserCount  string `json:"convert_user_count"`
		ClickRate         string `json:"click_rate"`
		ClickConvertRate  string `json:"click_convert_rate"`
		ExposeConvertRate string `json:"expose_convert_rate"`
		OrderCount        string `json:"order_count"`
		SalesVolume       string `json:"sales_volume"`
		AuthorName        string `json:"author_name"`
		CoverURL          string `json:"video_image"`
		PublishTime       string `json:"publish_time"`
	}

	type response struct {
		Data struct {
			VideoList []videoItem `json:"video_list"`
		} `json:"data"`
	}

	var resp response
	_ = json.Unmarshal([]byte(body), &resp)

	videos := make([]VideoAnalysis, 0, len(resp.Data.VideoList))
	for _, item := range resp.Data.VideoList {
		if item.VideoId == "" {
			continue
		}
		video := VideoAnalysis{
			ProductID:         product.ProductID,
			ProductName:       product.ProductName,
			VideoId:           item.VideoId,
			VideoDesc:         item.VideoDesc,
			AuthorName:        item.AuthorName,
			CoverURL:          item.CoverURL,
			PublishTime:       item.PublishTime,
			SourceURL:         fmt.Sprintf("https://www.douyin.com/video/%s", item.VideoId),
			PaymentAmount:     item.PaymentAmount,
			ExposeUserCount:   item.ExposeUserCount,
			ClickUserCount:    item.ClickUserCount,
			ConvertUserCount:  item.ConvertUserCount,
			ClickRate:         item.ClickRate,
			ClickConvertRate:  item.ClickConvertRate,
			ExposeConvertRate: item.ExposeConvertRate,
			OrderCount:        item.OrderCount,
			SalesVolume:       item.SalesVolume,
		}
		videos = append(videos, video)
	}
	if len(videos) > 0 {
		return videos
	}

	if videos := parseGenericFlowMoreRows(body, product); len(videos) > 0 {
		return videos
	}

	// 平台改版导致字段层级变化时，从该商品专属的浏览器响应文本中
	// 兜底提取真实 video_id / aweme_id，只接受纯数字 ID。
	for _, identity := range extractAllVideoIdentitiesFromText(body) {
		videos = append(videos, VideoAnalysis{
			ProductID:   product.ProductID,
			ProductName: product.ProductName,
			VideoId:     identity.VideoId,
			VideoDesc:   product.ProductName,
			SourceURL:   identity.SourceURL,
		})
	}

	return videos
}

func parseGenericFlowMoreRows(body string, product ProductInfo) []VideoAnalysis {
	decoder := json.NewDecoder(bytes.NewReader([]byte(body)))
	decoder.UseNumber()
	var root any
	if err := decoder.Decode(&root); err != nil {
		return nil
	}

	rows := candidateVideoRows(root)
	videos := make([]VideoAnalysis, 0, len(rows))
	seen := make(map[string]struct{})
	for _, row := range rows {
		videoID := normalizeVideoIDCandidate(firstRecursiveString(row,
			"video_id", "aweme_id", "aweme_id_str", "item_id", "item_id_str",
		))
		if videoID == "" {
			for _, identity := range extractAllVideoIdentitiesFromText(marshalCompactString(row)) {
				videoID = identity.VideoId
				break
			}
		}
		if videoID == "" {
			continue
		}
		if _, ok := seen[videoID]; ok {
			continue
		}
		seen[videoID] = struct{}{}

		videoDesc := firstRecursiveString(row,
			"video_title", "video_desc", "item_title", "title", "desc", "content",
		)
		if strings.TrimSpace(videoDesc) == "" {
			videoDesc = product.ProductName
		}

		videos = append(videos, VideoAnalysis{
			ProductID:         product.ProductID,
			ProductName:       product.ProductName,
			VideoId:           videoID,
			VideoDesc:         videoDesc,
			AuthorName:        firstRecursiveString(row, "author_nick_name", "author_name", "nick_name", "nickname"),
			CoverURL:          firstRecursiveString(row, "video_image", "cover_url", "cover", "image_url", "avatar_url"),
			PublishTime:       firstRecursiveString(row, "publish_time", "create_time", "release_time"),
			SourceURL:         "https://www.douyin.com/video/" + videoID,
			PaymentAmount:     firstMetricString(row, "pay_amt", "payment_amount", "gmv"),
			ExposeUserCount:   firstMetricString(row, "show_uv", "expose_user_count", "exposure_user_count"),
			ClickUserCount:    firstMetricString(row, "dh_product_click_uv", "click_user_count", "product_click_user_count"),
			ConvertUserCount:  firstMetricString(row, "pay_ucnt", "convert_user_count"),
			ClickRate:         firstMetricString(row, "click_ratio", "click_rate"),
			ClickConvertRate:  firstMetricString(row, "click_conversion", "click_convert_rate"),
			ExposeConvertRate: firstMetricString(row, "show_conversion", "expose_convert_rate"),
			OrderCount:        firstMetricString(row, "pay_cnt", "order_count"),
			SalesVolume:       firstMetricString(row, "pay_combo_cnt", "sales_volume"),
		})
	}
	return videos
}

func candidateVideoRows(root any) []any {
	rows := make([]any, 0)
	if rootMap, ok := root.(map[string]any); ok {
		if data, exists := rootMap["data"]; exists {
			switch typed := data.(type) {
			case []any:
				return typed
			case map[string]any:
				for _, key := range []string{"video_list", "list", "items", "rows", "result"} {
					if list, ok := typed[key].([]any); ok {
						return list
					}
				}
			}
		}
	}
	collectLikelyVideoRows(root, &rows, 0)
	return rows
}

func collectLikelyVideoRows(value any, rows *[]any, depth int) {
	if depth > 4 || len(*rows) >= 200 {
		return
	}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if containsRecursiveKey(item, "video_id", "aweme_id", "aweme_id_str", "item_id", "item_id_str") {
				*rows = append(*rows, item)
			} else {
				collectLikelyVideoRows(item, rows, depth+1)
			}
		}
	case map[string]any:
		for _, item := range typed {
			collectLikelyVideoRows(item, rows, depth+1)
		}
	}
}

func containsRecursiveKey(value any, keys ...string) bool {
	return firstRecursiveString(value, keys...) != ""
}

func firstRecursiveString(value any, keys ...string) string {
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[strings.ToLower(key)] = struct{}{}
	}
	return firstRecursiveStringBySet(value, keySet, 0)
}

func firstRecursiveStringBySet(value any, keySet map[string]struct{}, depth int) string {
	if depth > 8 {
		return ""
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if _, ok := keySet[strings.ToLower(key)]; ok {
				if candidate := stringFromJSONValue(child); strings.TrimSpace(candidate) != "" {
					return candidate
				}
			}
		}
		keys := sortedMapKeys(typed)
		for _, key := range keys {
			if candidate := firstRecursiveStringBySet(typed[key], keySet, depth+1); strings.TrimSpace(candidate) != "" {
				return candidate
			}
		}
	case []any:
		for _, child := range typed {
			if candidate := firstRecursiveStringBySet(child, keySet, depth+1); strings.TrimSpace(candidate) != "" {
				return candidate
			}
		}
	}
	return ""
}

func stringFromJSONValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		if math.Trunc(typed) == typed {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	case nil:
		return ""
	default:
		return ""
	}
}

func normalizeVideoIDCandidate(candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if numericVideoID.MatchString(candidate) {
		return candidate
	}
	return ""
}

func marshalCompactString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func firstMetricString(value any, keys ...string) string {
	keySet := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		keySet[strings.ToLower(key)] = struct{}{}
	}
	metric := firstMetricValueBySet(value, keySet, 0)
	return formatGenericMetric(metric)
}

func firstMetricValueBySet(value any, keySet map[string]struct{}, depth int) any {
	if depth > 8 {
		return nil
	}
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if _, ok := keySet[strings.ToLower(key)]; ok {
				return child
			}
		}
		keys := sortedMapKeys(typed)
		for _, key := range keys {
			if candidate := firstMetricValueBySet(typed[key], keySet, depth+1); candidate != nil {
				return candidate
			}
		}
	case []any:
		for _, child := range typed {
			if candidate := firstMetricValueBySet(child, keySet, depth+1); candidate != nil {
				return candidate
			}
		}
	}
	return nil
}

func formatGenericMetric(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case map[string]any:
		if extra := stringFromJSONValue(typed["extra_value"]); extra != "" {
			return extra
		}
		unit := stringFromJSONValue(typed["unit"])
		if valueMap, ok := typed["value"].(map[string]any); ok {
			if lower, lowerOK := numberFromJSONValue(valueMap["lower"]); lowerOK {
				if upper, upperOK := numberFromJSONValue(valueMap["upper"]); upperOK {
					return (MetricField{
						Value: MetricValue{Lower: lower, Upper: upper},
						Unit:  unit,
					}).FormatRange()
				}
			}
		}
		if valueNode, exists := typed["value"]; exists {
			return formatGenericMetric(valueNode)
		}
		if lower, lowerOK := numberFromJSONValue(typed["lower"]); lowerOK {
			if upper, upperOK := numberFromJSONValue(typed["upper"]); upperOK {
				return (MetricField{
					Value: MetricValue{Lower: lower, Upper: upper},
					Unit:  unit,
				}).FormatRange()
			}
		}
	}
	return ""
}

func numberFromJSONValue(value any) (float64, bool) {
	switch typed := value.(type) {
	case json.Number:
		number, err := typed.Float64()
		return number, err == nil
	case float64:
		return typed, true
	case string:
		number, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return number, err == nil
	default:
		return 0, false
	}
}
