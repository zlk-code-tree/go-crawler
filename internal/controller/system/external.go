package system

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
)

// ExternalController 外链菜单控制器
// 路径前缀：/system/external

type ExternalController struct{}

func NewExternalController() *ExternalController {
	return &ExternalController{}
}

// ExternalLink 外链页面 GET /system/external/{code}
func (c *ExternalController) ExternalLink(r *ghttp.Request) {
	// 获取菜单编码
	code := r.Get("code").String()
	if code == "" {
		r.Response.Write("菜单编码不能为空")
		return
	}

	// 查询菜单的外链地址
	var extUrl string
	_, err := g.DB().Model("sys_permission").
		Where("code", code).
		Where("type", 1).   // 菜单类型
		Where("status", 1). // 启用状态
		Value("ext_url", &extUrl)

	if err != nil || extUrl == "" {
		r.Response.Write("外链菜单不存在或未配置外链地址")
		return
	}

	// 渲染外链页面模板
	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>外部链接</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #f0f2f5;
        }
        .external-container {
            display: flex;
            flex-direction: column;
            height: 100vh;
        }
        .external-header {
            background: #fff;
            padding: 16px 24px;
            border-bottom: 1px solid #e8e8e8;
            box-shadow: 0 1px 4px rgba(0, 0, 0, 0.1);
        }
        .external-title {
            font-size: 16px;
            font-weight: 600;
            color: #262626;
            display: flex;
            align-items: center;
            gap: 8px;
        }
        .external-url {
            font-size: 12px;
            color: #8c8c8c;
            margin-top: 4px;
            word-break: break-all;
        }
        .external-content {
            flex: 1;
            position: relative;
        }
        .external-iframe {
            width: 100%;
            height: 100%;
            border: none;
            background: #fff;
        }
        .external-loading {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            text-align: center;
            color: #8c8c8c;
        }
        .external-error {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            text-align: center;
            color: #ff4d4f;
        }
    </style>
</head>
<body>
    <div class="external-container">
        <div class="external-header">
            <div class="external-title">
                <span>🌐 外部链接</span>
            </div>
            <div class="external-url">
                目标地址：<span id="target-url">` + extUrl + `</span>
            </div>
        </div>
        <div class="external-content">
            <iframe 
                class="external-iframe" 
                src="` + extUrl + `" 
                id="external-frame"
                onload="hideLoading()"
                onerror="showError()"
                sandbox="allow-same-origin allow-scripts allow-popups allow-forms"
                referrerpolicy="no-referrer-when-downgrade">
            </iframe>
            <div id="loading" class="external-loading">
                <div>正在加载外部页面...</div>
            </div>
            <div id="error" class="external-error" style="display: none;">
                <div>页面加载失败，请检查网络连接或目标地址</div>
            </div>
        </div>
    </div>
    <script>
        function hideLoading() {
            document.getElementById('loading').style.display = 'none';
        }
        
        function showError() {
            document.getElementById('loading').style.display = 'none';
            document.getElementById('error').style.display = 'block';
        }
        
        // 设置 iframe 标题
        document.getElementById('external-frame').onload = function() {
            try {
                var iframeDoc = this.contentDocument || this.contentWindow.document;
                var title = iframeDoc.title || '外部页面';
                document.querySelector('.external-title span').textContent = title;
            } catch (e) {
                // 跨域限制，无法获取标题
            }
        };
        
        // 安全处理：防止 iframe 中的页面跳转顶层窗口
        window.addEventListener('message', function(e) {
            if (e.data && e.data.type === 'navigation') {
                // 可以在这里处理来自 iframe 的导航请求
                console.log('收到导航请求:', e.data);
            }
        });
    </script>
</body>
</html>`

	r.Response.Write(html)
}

// ExternalLinkList 获取外链菜单列表 GET /system/external/list
func (c *ExternalController) ExternalLinkList(r *ghttp.Request) {
	// 查询所有外链菜单
	result, err := g.DB().Model("sys_permission").
		Fields("code", "name", "ext_url", "icon").
		Where("type", 1).          // 菜单类型
		Where("status", 1).        // 启用状态
		Where("ext_url != ?", ""). // 有外链地址
		OrderAsc("sort").
		All()

	if err != nil {
		r.Response.WriteJsonExit(g.Map{"code": 500, "data": nil, "msg": "查询失败"})
		return
	}

	var externalMenus []g.Map
	for _, row := range result {
		externalMenus = append(externalMenus, g.Map{
			"code":   row["code"].String(),
			"name":   row["name"].String(),
			"extUrl": row["ext_url"].String(),
			"icon":   row["icon"].String(),
			"path":   "/external/" + row["code"].String(),
		})
	}

	r.Response.WriteJsonExit(g.Map{"code": 0, "data": externalMenus, "msg": "ok"})
}
