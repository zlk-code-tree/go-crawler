-- 清理 finance_daily_stats 重复数据并添加唯一索引
-- 保留每组重复的最小id
DELETE t1 FROM finance_daily_stats t1
INNER JOIN finance_daily_stats t2
WHERE t1.id > t2.id
  AND t1.stat_date = t2.stat_date
  AND t1.company_id = t2.company_id
  AND t1.department_id = t2.department_id
  AND t1.trader_id = t2.trader_id
  AND t1.account_id = t2.account_id
  AND t1.platform = t2.platform;

-- 添加唯一索引
ALTER TABLE finance_daily_stats
ADD UNIQUE INDEX uk_stat (stat_date, company_id, department_id, trader_id, account_id, platform);
