# 店铺管理 & 抖音订单 API 接口文档

## 基础信息

**Base URL:** `http://your-domain/api`

**认证方式:** Bearer Token (JWT)

**请求头:**
```
Authorization: Bearer {token}
Content-Type: application/json
```

---

## 一、店铺管理接口

### 1.1 店铺列表查询

**接口地址:** `GET /api/system/shop/list`

**请求参数:**
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | int | 是 | 页码，从1开始 |
| pageSize | int | 是 | 每页数量，建议10-100 |
| keyword | string | 否 | 关键词搜索（店铺UID、抖音号名称、抖音号、运营组长） |

**请求示例:**
```javascript
GET /api/system/shop/list?page=1&pageSize=10&keyword=测试店铺
```

**响应示例:**
```json
{
  "code": 0,
  "msg": "ok",
  "total": 100,
  "data": [
    {
      "id": 1,
      "uid": "shop_001",
      "douyinName": "测试店铺",
      "douyinId": "dy123456",
      "operationLeader": "张三",
      "accountOwnership": "自营",
      "shopPlatform": "抖音",
      "createdAt": "2024-01-01 10:00:00",
      "updatedAt": "2024-01-01 10:00:00"
    }
  ]
}
```

---

### 1.2 店铺详情查询

**接口地址:** `GET /api/system/shop/detail`

**请求参数:**
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | int | 是 | 店铺ID |

**请求示例:**
```javascript
GET /api/system/shop/detail?id=1
```

**响应示例:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "id": 1,
    "uid": "shop_001",
    "douyinName": "测试店铺",
    "douyinId": "dy123456",
    "operationLeader": "张三",
    "accountOwnership": "自营",
    "shopPlatform": "抖音",
    "createdAt": "2024-01-01 10:00:00",
    "updatedAt": "2024-01-01 10:00:00"
  }
}
```

---

### 1.3 保存店铺（新增/编辑）

**接口地址:** `POST /api/system/shop/save`

**请求参数:**
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | int | 否 | 店铺ID，不传或为0表示新增，传值表示编辑 |
| uid | string | 是 | 店铺唯一标识 |
| douyinName | string | 是 | 抖音号名称 |
| douyinId | string | 否 | 抖音号 |
| operationLeader | string | 是 | 运营组长 |
| accountOwnership | string | 否 | 账号归属 |
| shopPlatform | string | 是 | 店铺平台（如：抖音、淘宝、京东） |

**请求示例:**
```javascript
POST /api/system/shop/save
Content-Type: application/json

{
  "id": 0,
  "uid": "shop_002",
  "douyinName": "新店铺",
  "douyinId": "dy789012",
  "operationLeader": "李四",
  "accountOwnership": "代运营",
  "shopPlatform": "抖音"
}
```

**响应示例:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "id": 2
  }
}
```

---

### 1.4 删除店铺

**接口地址:** `POST /api/system/shop/delete`

**请求参数:**
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| id | int | 是 | 店铺ID |

**请求示例:**
```javascript
POST /api/system/shop/delete
Content-Type: application/json

{
  "id": 1
}
```

**响应示例:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": null
}
```

---

### 1.5 导出店铺Excel

**接口地址:** `GET /api/system/shop/export`

**请求参数:**
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| keyword | string | 否 | 关键词搜索 |

**请求示例:**
```javascript
GET /api/system/shop/export?keyword=测试
```

**响应:** 直接下载Excel文件

**Excel表头:**
- uid（店铺唯一标识）
- douyin_name（抖音号名称）
- douyin_id（抖音号）
- operation_leader（运营组长）
- account_ownership（账号归属）
- shop_platform（店铺平台）

---

### 1.6 导入店铺Excel

**接口地址:** `POST /api/system/shop/import`

**请求参数:**
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| file | file | 是 | Excel文件（.xlsx格式，最大50MB） |

**请求示例:**
```javascript
POST /api/system/shop/import
Content-Type: multipart/form-data

file: [Excel文件]
```

**Excel格式要求:**
- 第一行必须是表头：uid, douyin_name, douyin_id, operation_leader, account_ownership, shop_platform
- 必填字段：uid, douyin_name, operation_leader, shop_platform
- 支持流式处理，每1000条批量入库
- 自动跳过已存在的UID

**响应示例:**
```json
{
  "code": 0,
  "msg": "导入完成：成功48500条，跳过1200条，失败300条",
  "data": {
    "totalRows": 50000,
    "successCount": 48500,
    "skipCount": 1200,
    "failCount": 300,
    "errors": [
      "第5行: uid不能为空",
      "第12行: uid=shop_001 已存在"
    ]
  }
}
```

---

## 二、抖音订单接口

### 2.1 订单列表查询

**接口地址:** `GET /api/system/order/list`

**请求参数:**
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | int | 是 | 页码，从1开始 |
| pageSize | int | 是 | 每页数量，建议10-100 |
| keyword | string | 否 | 关键词搜索（订单号、商品名称） |
| shopName | string | 否 | 店铺名称筛选 |
| orderStatus | string | 否 | 订单状态筛选 |
| startTime | string | 否 | 开始时间（格式：2024-01-01 00:00:00） |
| endTime | string | 否 | 结束时间（格式：2024-01-31 23:59:59） |

**请求示例:**
```javascript
GET /api/system/order/list?page=1&pageSize=20&shopName=测试店铺&orderStatus=已完成&startTime=2024-01-01 00:00:00&endTime=2024-01-31 23:59:59
```

**响应示例:**
```json
{
  "code": 0,
  "msg": "ok",
  "total": 5000,
  "data": [
    {
      "id": 1,
      "shopName": "测试店铺",
      "mainOrderNo": "M202401010001",
      "subOrderNo": "S202401010001",
      "productName": "测试商品",
      "productSpec": "红色/L",
      "productNum": 2,
      "productId": "P123456",
      "productPrice": 99.00,
      "orderPayAmount": 198.00,
      "shippingFee": 10.00,
      "totalDiscount": 20.00,
      "orderStatus": "已完成",
      "orderSubmitTime": "2024-01-01 10:00:00",
      "payFinishTime": "2024-01-01 10:05:00",
      "orderFinishTime": "2024-01-05 10:00:00",
      "receiverName": "张三",
      "receiverPhone": "13800138000",
      "province": "广东省",
      "city": "深圳市",
      "district": "南山区",
      "talentName": "达人昵称",
      "flowSource": "短视频",
      "afterSaleStatus": "无售后"
    }
  ]
}
```

---

### 2.2 导出订单Excel

**接口地址:** `GET /api/system/order/export`

**请求参数:**
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| keyword | string | 否 | 关键词搜索 |
| shopName | string | 否 | 店铺名称筛选 |
| orderStatus | string | 否 | 订单状态筛选 |
| startTime | string | 否 | 开始时间 |
| endTime | string | 否 | 结束时间 |

**请求示例:**
```javascript
GET /api/system/order/export?shopName=测试店铺&orderStatus=已完成&startTime=2024-01-01 00:00:00&endTime=2024-01-31 23:59:59
```

**响应:** 直接下载Excel文件

**Excel包含62个字段（完整订单信息）:**
店铺、主订单编号、子订单编号、选购商品、商品规格、商品数量、商品ID、商家编码、商品单价、订单应付金额、运费、优惠总金额、平台优惠、商家优惠、达人优惠、商家改价、支付优惠、红包抵扣、支付方式、手续费、收件人、收件人手机号、省、市、区、街道、详细地址、是否修改过地址、地址修改时段、买家留言、订单提交时间、旗帜颜色、商家备注、订单完成时间、支付完成时间、APP渠道、流量来源、订单状态、承诺发货时间、订单类型、鲁班落地页ID、达人ID、达人昵称、所属门店ID、售后状态、取消原因、预约发货时间、仓库ID、仓库名称、是否安心购、广告渠道、流量类型、流量体裁、流量渠道、发货主体、发货主体明细、发货时间、降价类优惠、平台实际承担优惠金额、商家实际承担优惠金额、达人实际承担优惠金额、预计送达时间

---

### 2.3 导入订单Excel

**接口地址:** `POST /api/system/order/import`

**请求参数:**
| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| file | file | 是 | Excel文件（.xlsx格式，最大100MB） |

**请求示例:**
```javascript
POST /api/system/order/import
Content-Type: multipart/form-data

file: [Excel文件]
```

**Excel格式要求:**
- 第一行必须是中文表头（与导出格式一致）
- 必填字段：店铺、子订单编号、选购商品、商品单价、订单应付金额、订单状态
- 支持流式处理，每1000条批量入库
- 自动跳过已存在的子订单号
- 支持10万+行数据导入
- 超时时间：10分钟

**响应示例:**
```json
{
  "code": 0,
  "msg": "导入完成：成功98500条，跳过1200条，失败300条",
  "data": {
    "totalRows": 100000,
    "successCount": 98500,
    "skipCount": 1200,
    "failCount": 300,
    "errors": [
      "第5行: 店铺不能为空",
      "第12行: 子订单编号不能为空",
      "第25行: 子订单编号=S202401010001 已存在"
    ]
  }
}
```

---

## 三、错误码说明

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 400 | 请求参数错误 |
| 401 | 未授权，需要登录 |
| 403 | 无权限访问 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

---

## 四、前端调用示例

### 4.1 使用 Axios

```javascript
import axios from 'axios';

// 配置基础URL和拦截器
const api = axios.create({
  baseURL: 'http://your-domain/api',
  timeout: 600000, // 10分钟超时（用于导入导出）
});

// 请求拦截器 - 添加token
api.interceptors.request.use(config => {
  const token = localStorage.getItem('token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// 1. 查询店铺列表
export const getShopList = (params) => {
  return api.get('/system/shop/list', { params });
};

// 2. 保存店铺
export const saveShop = (data) => {
  return api.post('/system/shop/save', data);
};

// 3. 删除店铺
export const deleteShop = (id) => {
  return api.post('/system/shop/delete', { id });
};

// 4. 导出店铺
export const exportShop = (params) => {
  return api.get('/system/shop/export', {
    params,
    responseType: 'blob', // 重要：设置响应类型为blob
  });
};

// 5. 导入店铺
export const importShop = (file) => {
  const formData = new FormData();
  formData.append('file', file);
  return api.post('/system/shop/import', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
};

// 6. 查询订单列表
export const getOrderList = (params) => {
  return api.get('/system/order/list', { params });
};

// 7. 导出订单
export const exportOrder = (params) => {
  return api.get('/system/order/export', {
    params,
    responseType: 'blob',
  });
};

// 8. 导入订单
export const importOrder = (file) => {
  const formData = new FormData();
  formData.append('file', file);
  return api.post('/system/order/import', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
};
```

### 4.2 使用示例

```javascript
// 查询店铺列表
const fetchShops = async () => {
  try {
    const res = await getShopList({
      page: 1,
      pageSize: 10,
      keyword: '测试'
    });
    console.log(res.data);
  } catch (error) {
    console.error('查询失败', error);
  }
};

// 导入店铺
const handleImport = async (file) => {
  try {
    const res = await importShop(file);
    if (res.data.code === 0) {
      alert(`导入成功：${res.data.data.successCount}条`);
    }
  } catch (error) {
    console.error('导入失败', error);
  }
};

// 导出店铺
const handleExport = async () => {
  try {
    const res = await exportShop({ keyword: '测试' });
    // 创建下载链接
    const url = window.URL.createObjectURL(new Blob([res.data]));
    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', `shop_export_${Date.now()}.xlsx`);
    document.body.appendChild(link);
    link.click();
    link.remove();
  } catch (error) {
    console.error('导出失败', error);
  }
};
```

---

## 五、注意事项

1. **认证要求:** 所有接口都需要在请求头中携带有效的JWT Token
2. **超时设置:** 导入导出接口建议设置较长的超时时间（10分钟）
3. **文件大小限制:** 
   - 店铺导入：最大50MB
   - 订单导入：最大100MB
4. **Excel格式:** 
   - 必须是.xlsx格式
   - 表头必须与文档要求一致
   - 导入和导出的表头格式相同，可以导出后修改再导入
5. **批量处理:** 导入接口采用流式处理+批量入库，支持大数据量导入
6. **错误处理:** 导入失败时会返回详细的错误信息，包括具体行号和原因
