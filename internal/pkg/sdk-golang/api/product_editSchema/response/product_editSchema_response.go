package product_editSchema_response

import (
	"doudian.com/open/sdk_golang/core"
)

type ProductEditSchemaResponse struct {
	doudian_sdk.BaseDoudianOpApiResponse
	Data *ProductEditSchemaData `json:"data"`
}
type ProductEditSchemaData struct {
}
