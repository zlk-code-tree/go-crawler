#!/usr/bin/env python3
"""创建 account 表唯一索引"""
import pymysql

conn = pymysql.connect(
    host='rm-2zeyw894ggx6yj3i2yo.mysql.rds.aliyuncs.com',
    port=3306, user='ly_admin',
    password='4qPbP%dd#aQs8wR',
    database='ly_crawler', charset='utf8mb4'
)
c = conn.cursor()

# 检查索引
c.execute("SHOW INDEX FROM account WHERE Key_name = 'uk_platform_account_name'")
if c.fetchone():
    print('唯一索引已存在，无需重复创建')
else:
    c.execute("ALTER TABLE account ADD UNIQUE KEY uk_platform_account_name (platform, account_name) COMMENT '平台+账号名称唯一'")
    conn.commit()
    print('唯一索引 uk_platform_account_name 创建成功')

# 验证
c.execute("SELECT status, COUNT(*) FROM account GROUP BY status")
print('\n最终数据:')
for r in c.fetchall():
    label = "正常" if r[0] == 1 else "已删除"
    print(f'  status={r[0]} ({label}): {r[1]} 条')

c.execute("SELECT platform, COUNT(*) FROM account WHERE status=1 GROUP BY platform")
print('\n正常账号平台分布:')
for r in c.fetchall():
    print(f'  {r[0]}: {r[1]}')

c.close()
conn.close()
print('\n完成')
