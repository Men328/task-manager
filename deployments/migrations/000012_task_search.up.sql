-- =============================================================================
-- 000012_task_search
-- Tìm kiếm task theo tiêu đề + mô tả, không phân biệt hoa thường và dấu
-- ("bao cao" khớp "Báo cáo", "do an" khớp "Đồ án"), khớp một phần.
-- Phụ thuộc: 000002_init_task
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS unaccent;

-- unaccent() chỉ STABLE (phụ thuộc search_path/từ điển) nên không dùng trực tiếp
-- trong index được; bọc lại với từ điển ghi rõ schema để khai báo IMMUTABLE.
-- Repository dùng đúng hàm này cho cả cột lẫn chuỗi tìm kiếm.
CREATE OR REPLACE FUNCTION task.f_search_text(value text)
RETURNS text
LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
AS $$
  SELECT lower(public.unaccent('public.unaccent'::regdictionary, value))
$$;

-- Index trigram trên tiêu đề + mô tả đã chuẩn hoá: phục vụ LIKE '%...%' và similarity().
CREATE INDEX IF NOT EXISTS ix_tasks_search_trgm
    ON task.TASKS
    USING gin (task.f_search_text(title || ' ' || COALESCE(description, '')) gin_trgm_ops)
    WHERE deleted_at IS NULL;
