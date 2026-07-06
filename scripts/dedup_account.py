#!/usr/bin/env python3
"""account 表去重：保留每组 platform+account_name 中 id 最小的记录，其余软删除(status=0)"""
import pymysql

conn = pymysql.connect(
    host='rm-2zeyw894ggx6yj3i2yo.mysql.rds.aliyuncs.com',
    port=3306, user='ly_admin',
    password='4qPbP%dd#aQs8wR',
    database='ly_crawler', charset='utf8mb4'
)
c = conn.cursor()

# 1. 统计重复情况
c.execute("""
    SELECT COUNT(*) as dup_groups, SUM(cnt - 1) as to_delete
    FROM (
        SELECT platform, account_name, COUNT(*) as cnt
        FROM account
        GROUP BY platform, account_name
        HAVING cnt > 1
    ) t
""")
row = c.fetchone()
dup_groups, to_delete = row[0], row[1]
print(f"重复组数: {dup_groups}")
print(f"待去重记录数: {to_delete}")

if to_delete == 0:
    print("无需去重")
else:
    # 2. 找出每组重复数据中需要标记删除的 id（保留最小的）
    print("正在去重...")
    c.execute("""
        UPDATE account a
        INNER JOIN (
            SELECT platform, account_name, MIN(id) as keep_id
            FROM account
            GROUP BY platform, account_name
            HAVING COUNT(*) > 1
        ) k ON a.platform = k.platform AND a.account_name = k.account_name
        SET a.status = 0, a.updated_at = NOW()
        WHERE a.id > k.keep_id AND a.status = 1
    """)
    conn.commit()
    affected = c.rowcount
    print(f"  已标记 {affected} 条为软删除 (status=0)")

# 3. 验证：确认无重复
c.execute("""
    SELECT platform, account_name, COUNT(*) as cnt
    FROM account
    WHERE status = 1
    GROUP BY platform, account_name
    HAVING cnt > 1
    LIMIT 5
""")
remaining = c.fetchall()
if remaining:
    print(f"\n⚠️  仍有 {len(remaining)} 组重复（status=1）:")
    for r in remaining:
        print(f"  {r[0]}/{r[1]}: {r[2]}条")
else:
    print("\n✅ 重复数据已清理完毕（status=1 的记录中无重复）")

# 4. 统计最终状态
c.execute("SELECT status, COUNT(*) FROM account GROUP BY status")
print("\n最终状态分布:")
for r in c.fetchall():
    label = "正常" if r[0] == 1 else "已删除" if r[0] == 0 else str(r[0])
    print(f"  status={r[0]} ({label}): {r[1]} 条")

c.close()
conn.close()
print("\n完成")
