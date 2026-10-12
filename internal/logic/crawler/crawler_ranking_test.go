package crawler

import "testing"

func TestParseProductRankResponse(t *testing.T) {
	response := `{"data":{"data_result":[{"product_info":{"rank":1,"id":"product-1","name":"测试商品","image_url":"https://example.test/product.png","shop_list":[{"shop_id":"shop-1","shop_name":"测试店铺"}]},"pay_amt":{"value_range":[{"value":100,"unit":"yuan"}]}}]}}`
	products := parseProductRankResponse(response)
	if len(products) != 1 {
		t.Fatalf("expected one product, got %+v", products)
	}
	got := products[0]
	if got.Rank != 1 || got.ProductID != "product-1" || got.ShopID != "shop-1" || got.ProductName != "测试商品" || got.ImageURL != "https://example.test/product.png" {
		t.Fatalf("ranking response lost product identity: %+v", got)
	}
	if products := parseProductRankResponse(`{"data":`); len(products) != 0 {
		t.Fatalf("malformed response yielded products: %+v", products)
	}
}
