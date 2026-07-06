#!/usr/bin/env python3
"""为 account 表补全缺失的列和索引"""
import pymysql, sys

conn = pymysql.connect(
    host='rm-2zeyw894ggx6yj3i2yo.mysql.rds.aliyuncs.com',
    port=3306, user='ly_admin',
    password='4qPbP%dd#aQs8wR',
    database='ly_crawler', charset='utf8mb4'
)
c = conn.cursor()

# 1. 检查已有列
c.execute("SELECT COLUMN_NAME FROM information_schema.columns WHERE table_schema='ly_crawler' AND table_name='account'")
existing_cols = {r[0] for r in c.fetchall()}
print(f"account 表现有列: {sorted(existing_cols)}")

# 2. 添加缺失的时间列
additions = [
    ('created_at', "ALTER TABLE account ADD COLUMN created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间'"),
    ('updated_at', "ALTER TABLE account ADD COLUMN updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间'"),
]
for col, ddl in additions:
    if col not in existing_cols:
        c.execute(ddl)
        conn.commit()
        print(f"  ✅ {col} 添加成功")
    else:
        print(f"  ⏭️  {col} 已存在")

# 3. 检查并添加唯一索引 (platform, account_name)
c.execute("SHOW INDEX FROM account WHERE Key_name = 'uk_platform_account_name'")
if c.fetchone():
    print(f"  ⏭️  uk_platform_account_name 索引已存在")
else:
    # 先检查是否有重复数据会阻止创建唯一索引
    c.execute("""
        SELECT platform, account_name, COUNT(*) as cnt
        FROM account
        GROUP BY platform, account_name
        HAVING cnt > 1
        LIMIT 1
    """)
    dup = c.fetchone()
    if dup:
        print(f"  ⚠️  存在重复数据 (platform={dup[0]}, name={dup[1]}, cnt={dup[2]})，无法创建唯一索引")
        print(f"  请先去重后再创建唯一索引")
        print(f"  去重SQL: 见 scripts/dedup_account.sql")
    else:
        try:
            c.execute("ALTER TABLE account ADD UNIQUE KEY uk_platform_account_name (platform, account_name)")
            conn.commit()
            print(f"  ✅ uk_platform_account_name 唯一索引添加成功")
        except pymysql.Error as e:
            print(f"  ❌ 添加唯一索引失败: {e}")

# 4. 验证最终表结构
print("\n  最终表结构:")
c.execute("SHOW COLUMNS FROM account")
for r in c.fetchall():
    print(f"    {r[0]:25s} {r[1]:25s} Null={r[2]} Default={r[4]}")

c.execute("SHOW INDEX FROM account")
print("\n  索引:")
for r in c.fetchall():
    print(f"    {r[2]:30s} {r[4]:20s} unique={not r[1]}")

c.close()
conn.close()
print("\n完成")
