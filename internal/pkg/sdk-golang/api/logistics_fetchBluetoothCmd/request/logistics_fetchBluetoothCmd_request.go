package logistics_fetchBluetoothCmd_request

import (
	"doudian.com/open/sdk_golang/api/logistics_fetchBluetoothCmd/response"
	"doudian.com/open/sdk_golang/core"
	"encoding/json"
)

type LogisticsFetchBluetoothCmdRequest struct {
	doudian_sdk.BaseDoudianOpApiRequest
	Param *LogisticsFetchBluetoothCmdParam 
}
func (c *LogisticsFetchBluetoothCmdRequest) GetUrlPath() string{
	return "/logistics/fetchBluetoothCmd"
}


func New() *LogisticsFetchBluetoothCmdRequest{
	request := &LogisticsFetchBluetoothCmdRequest{
		Param: &LogisticsFetchBluetoothCmdParam{},
	}
	request.SetConfig(doudian_sdk.GlobalConfig)
	request.SetClient(doudian_sdk.DefaultDoudianOpApiClient)
	return request

}


func (c *LogisticsFetchBluetoothCmdRequest) Execute(accessToken *doudian_sdk.AccessToken) (*logistics_fetchBluetoothCmd_response.LogisticsFetchBluetoothCmdResponse, error){
	responseJson, err := c.GetClient().Request(c, accessToken)
	if err != nil {
		return nil, err
	}
	response := &logistics_fetchBluetoothCmd_response.LogisticsFetchBluetoothCmdResponse{}
	_ = json.Unmarshal([]byte(responseJson), response)
	return response, nil

}


func (c *LogisticsFetchBluetoothCmdRequest) GetParamObject() interface{}{
	return c.Param
}


func (c *LogisticsFetchBluetoothCmdRequest) GetParams() *LogisticsFetchBluetoothCmdParam{
	return c.Param
}


type WaybillAppliesItem struct {
	// 运单号
	TrackNo string `json:"track_no"`
	// 标准模板code。已支持的模板信息 https://bytedance.feishu.cn/sheets/VFmQsxJNth8t2OtWWDJca5BbnDb?sheet=p92gXy
	StdTemplateCode string `json:"std_template_code"`
	// 自定义区模板url。（传了cus_print_data后url和code其中一个必传）
	CusTemplateUrl *string `json:"cus_template_url"`
	// 自定义区模板code（目前仅支持模板设计器的自定义区模板）
	CusTemplateCode *string `json:"cus_template_code"`
	// 自定义区json数据
	CusPrintData *string `json:"cus_print_data"`
}
type RemotePrintConfig struct {
	// 是否需要打印物流商logo
	NeedTopLogo bool `json:"need_top_logo"`
	// 水平偏移量，单位mm
	HorizontalOffset *int32 `json:"horizontal_offset"`
	// 垂直偏移量，单位mm
	VerticalOffset *int32 `json:"vertical_offset"`
	// 打印方向。normal-正常；reverse-反转。
	PrintDirection *string `json:"print_direction"`
	// 打印机名称。查看已支持的打印机信息 https://bytedance.feishu.cn/sheets/VFmQsxJNth8t2OtWWDJca5BbnDb?sheet=44acf5
	PrinterName string `json:"printer_name"`
}
type LogisticsFetchBluetoothCmdParam struct {
	// 运单信息，一次最多请求200单。
	WaybillApplies []WaybillAppliesItem `json:"waybill_applies"`
	// 打印配置
	RemotePrintConfig *RemotePrintConfig `json:"remote_print_config"`
	// 打印机指令类型。CPCL；TSPL
	CmdType string `json:"cmd_type"`
	// 编码方式。origin：原串->字节数组->base64；gzip_byte：原串->字节数组->gzip压缩->base64
	CmdEncode string `json:"cmd_encode"`
}
