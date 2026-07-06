#!/usr/bin/env python3
"""为 account 表添加 created_at/updated_at 列"""
import pymysql, sys

conn = pymysql.connect(
    host='rm-2zeyw894ggx6yj3i2yo.mysql.rds.aliyuncs.com',
    port=3306, user='ly_admin',
    password='4qPbP%dd#aQs8wR',
    database='ly_crawler', charset='utf8mb4'
)
c = conn.cursor()

c.execute("SELECT COLUMN_NAME FROM information_schema.columns WHERE table_schema='ly_crawler' AND table_name='account' AND COLUMN_NAME IN ('created_at','updated_at')")
existing = {r[0] for r in c.fetchall()}
print(f"已有时间列: {existing}")

for col, ddl in [
    ('created_at', "ALTER TABLE account ADD COLUMN created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间'"),
    ('updated_at', "ALTER TABLE account ADD COLUMN updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'"),
]:
    if col not in existing:
        c.execute(ddl)
        conn.commit()
        print(f"  ✅ {col} 添加成功")
    else:
        print(f"  ⏭️  {col} 已存在，跳过")

c.close()
conn.close()
print("完成")
