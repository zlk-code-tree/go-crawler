#!/usr/bin/env python3
"""分析 account 表重复数据"""
import pymysql

conn = pymysql.connect(
    host='rm-2zeyw894ggx6yj3i2yo.mysql.rds.aliyuncs.com',
    port=3306,
    user='ly_admin',
    password='4qPbP%dd#aQs8wR',
    database='ly_crawler',
    charset='utf8mb4'
)
c = conn.cursor()

# 1. 表结构
print("=" * 80)
print("  表结构")
print("=" * 80)
c.execute('SHOW COLUMNS FROM account')
cols_list = []
for r in c.fetchall():
    cols_list.append(r[0])
    print(f"  {r[0]:25s} {r[1]:25s} Null={r[2]} Default={r[4]}")

# 2. 取一组重复样本，展示全部字段差异
print("\n" + "=" * 80)
print("  典型3重复记录的全部字段对比")
print("=" * 80)

c.execute("""
    SELECT *
    FROM account
    WHERE platform = 'shipinhao' AND account_name = '1-开心超人'
    ORDER BY id
""")
rows = c.fetchall()
for row in rows:
    print(f"\n  --- id={row[0]} ---")
    for i, col in enumerate(cols_list):
        print(f"    {col:20s}: {row[i]}")

# 3. 分类统计：字段差异类型
print("\n" + "=" * 80)
print("  277组重复中，哪些字段有差异")
print("=" * 80)

all_cols = ','.join([f'GROUP_CONCAT({c} ORDER BY id SEPARATOR "|||") as g_{c}' for c in cols_list])
c.execute(f"""
    SELECT platform, account_name, COUNT(*) as cnt,
           GROUP_CONCAT(id ORDER BY id) as ids,
           {all_cols}
    FROM account
    GROUP BY platform, account_name
    HAVING cnt > 1
""")
dup_groups = c.fetchall()

# 解析结果
total = len(dup_groups)
stats = {col: 0 for col in cols_list}

for g in dup_groups:
    # g[0]=platform, g[1]=name, g[2]=cnt, g[3]=ids, g[4:]=group_concat per column
    for i, col in enumerate(cols_list):
        if col in ('id', 'status'):
            continue
        vals = g[4 + i]
        if vals is None:
            continue
        val_list = vals.split('|||')
        # 去除空字符串视为相同
        cleaned = [v for v in val_list if v != '']
        if len(set(val_list)) > 1:
            stats[col] += 1

print(f"  重复组总数: {total}")
for col in cols_list:
    if col in ('id', 'status'):
        continue
    cnt = stats[col]
    bar = '█' * (cnt * 80 // total) if total > 0 else ''
    print(f"  {col:25s}: {cnt:4d}/{total} ({cnt*100//total if total else 0:3d}%) {bar}")

# 4. 完全相同（所有非id非时间字段都一样）
print("\n" + "=" * 80)
print("  各差异类别统计")
print("=" * 80)

any_diff = 0
no_diff = 0
only_nick_diff = 0
only_operator_diff = 0
only_uid_diff = 0

for g in dup_groups:
    diffs = []
    for i, col in enumerate(cols_list):
        if col in ('id', 'status'):
            continue
        vals = g[4 + i]
        if vals is None:
            continue
        val_list = vals.split('|||')
        cleaned = set(v for v in val_list if v != '')
        if len(set(val_list)) > 1 or len(cleaned) > 1:
            diffs.append(col)

    if len(diffs) == 0:
        no_diff += 1
    else:
        any_diff += 1

print(f"  所有字段完全一致（纯重复）: {no_diff}")
print(f"  有差异: {any_diff}")
print(f"  差异字段统计: {dict(sorted(stats.items(), key=lambda x: -x[1]))}")

# 5. 列出完全相同的部分样例
print("\n" + "=" * 80)
print("  完全一致（除id外）的重复样例（前20组）")
print("=" * 80)
# 手动检查：取 platform_nickname 全部相同的组
c.execute("""
    SELECT platform, account_name, COUNT(*) as cnt, GROUP_CONCAT(id ORDER BY id) as ids
    FROM account
    GROUP BY platform, account_name
    HAVING cnt > 1 AND COUNT(DISTINCT COALESCE(platform_nickname, '')) = 1
       AND COUNT(DISTINCT COALESCE(uid, '')) = 1
       AND COUNT(DISTINCT COALESCE(operator, '')) = 1
       AND COUNT(DISTINCT COALESCE(company_id, '')) = 1
    LIMIT 20
""")
for r in c.fetchall():
    print(f"  {r[0]}/{r[1]}: cnt={r[2]}, ids=[{r[3]}]")

c.close()
conn.close()
PYEOF