-- =====================================================
-- 数据库字符集和校对规则统一修复脚本
-- 功能：将所有相关表的字符集统一为 utf8mb4，校对规则统一为 utf8mb4_0900_ai_ci
-- 说明：utf8mb4_0900_ai_ci 是 MySQL 8.0+ 的默认校对规则，性能更好
-- =====================================================

-- 第一步：检查当前所有表的字符集和校对规则
-- 执行以下查询查看当前状态（保留查询，方便对比）
/*
SELECT 
    TABLE_NAME,
    CCSA.character_set_name AS character_set,
    TABLE_COLLATION
FROM information_schema.`TABLES` AS T,
     information_schema.`COLLATION_CHARACTER_SET_APPLICABILITY` AS CCSA
WHERE CCSA.collation_name = T.table_collation
  AND T.table_schema = DATABASE()
  AND T.TABLE_TYPE = 'BASE TABLE'
ORDER BY TABLE_NAME;
*/

-- 第二步：检查相关表的字段级别字符集（重点检查 JOIN 关联的字段）
-- 店铺表、商品表、订单表的字符集
/*
SELECT 
    TABLE_NAME,
    COLUMN_NAME,
    CHARACTER_SET_NAME,
    COLLATION_NAME
FROM information_schema.COLUMNS
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME IN ('shop', 'goods', 'order_info')
  AND DATA_TYPE IN ('varchar', 'char', 'text', 'mediumtext', 'longtext')
ORDER BY TABLE_NAME, COLUMN_NAME;
*/


-- =====================================================
-- 第三步：统一修复相关表的字符集和校对规则
-- =====================================================

-- 3.1 修复 shop 表（店铺管理）
ALTER TABLE `shop` 
    CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- 3.2 修复 goods 表（商品管理）
ALTER TABLE `goods` 
    CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- 3.3 修复 order_info 表（订单管理）
ALTER TABLE `order_info` 
    CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- 3.4 修复 order_receiver 表（订单收货人）
ALTER TABLE `order_receiver` 
    CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- 3.5 修复 order_address 表（订单地址）
ALTER TABLE `order_address` 
    CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;


-- =====================================================
-- 第四步：如果还有其他业务相关表，一并修复
-- 请根据实际情况取消注释执行
-- =====================================================

-- 系统相关表（如果需要）
-- ALTER TABLE `sys_user` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- ALTER TABLE `sys_dept` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- ALTER TABLE `sys_role` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
-- ALTER TABLE `sys_menu` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;


-- =====================================================
-- 第五步：验证修复结果
-- 执行以下查询确认所有表已统一
-- =====================================================
/*
SELECT 
    TABLE_NAME,
    TABLE_COLLATION,
    CASE 
        WHEN TABLE_COLLATION = 'utf8mb4_0900_ai_ci' THEN '✅ 正常'
        ELSE '❌ 异常'
    END AS status
FROM information_schema.`TABLES`
WHERE TABLE_SCHEMA = DATABASE()
  AND TABLE_NAME IN ('shop', 'goods', 'order_info', 'order_receiver', 'order_address')
ORDER BY TABLE_NAME;
*/


-- =====================================================
-- 补充说明：
-- 1. utf8mb4_0900_ai_ci 是 MySQL 8.0+ 默认校对规则
--    - 0900 表示 Unicode 9.0.0 版本
--    - ai 表示 accent-insensitive（口音不敏感）
--    - ci 表示 case-insensitive（大小写不敏感）
-- 
-- 2. 如果您的 MySQL 版本低于 8.0，请使用 utf8mb4_unicode_ci
--    将上面所有 SQL 中的 utf8mb4_0900_ai_ci 替换为 utf8mb4_unicode_ci
--
-- 3. CONVERT TO CHARACTER SET 会同时修改表和所有文本字段的字符集
-- =====================================================
