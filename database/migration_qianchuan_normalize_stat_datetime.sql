-- 统一千川报表日期格式，避免 `2026-06-30` 与 `2026-06-30 00:00:00` 被唯一键当作两天。
-- 执行前建议先确认不存在归一化后的重复键；当前业务插入逻辑已改为写入 YYYY-MM-DD。

UPDATE `qianchuan_advertiser_daily_report`
SET `stat_datetime` = LEFT(`stat_datetime`, 10)
WHERE LENGTH(`stat_datetime`) > 10;

UPDATE `qianchuan_ad_report`
SET `stat_datetime` = LEFT(`stat_datetime`, 10)
WHERE LENGTH(`stat_datetime`) > 10;
