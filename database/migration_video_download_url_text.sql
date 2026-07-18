-- 视频直链通常是带签名的临时 CDN URL，长度可能超过 2048。
-- 使用 TEXT 避免解析成功后因字段过短被误标为 failed。
ALTER TABLE `video_analysis_data`
  MODIFY COLUMN `download_url` TEXT NOT NULL;
