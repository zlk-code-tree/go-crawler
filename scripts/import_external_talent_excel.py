#!/usr/bin/env python3
from __future__ import annotations

import argparse
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Dict, Iterable, List, Tuple

import pymysql
import xlrd


DB_LINK_PATTERN = re.compile(
    r'link:\s*"(?P<user>[^:"]+):(?P<password>.+?)@tcp\((?P<host>[^:()]+):(?P<port>\d+)\)/(?P<database>[^"?]+)'
)


@dataclass
class ExternalTalentRow:
    team_name: str
    platform_type: str
    shop_name: str
    platform_shop_label: str
    uid: str
    account_name: str
    show_id: str
    link_code: str
    talent_name: str
    product_id: str
    goods_code: str
    customer_price: float
    online_commission_rate: float
    offline_commission_rate: float
    retain_rate: float
    remark: str
    status: int
    source_file: str
    source_sheet: str
    source_row_no: int

    def dedupe_key(self) -> Tuple[str, ...]:
        return (
            self.team_name,
            self.platform_type,
            self.shop_name,
            self.uid,
            self.account_name,
            self.show_id,
            self.link_code,
            self.product_id,
            self.goods_code,
        )

    def exact_row_key(self) -> Tuple[str, ...]:
        return (
            self.team_name,
            self.platform_shop_label,
            self.uid,
            self.account_name,
            self.show_id,
            self.link_code,
            self.product_id,
            self.goods_code,
            format_decimal(self.customer_price),
            format_decimal(self.online_commission_rate),
            format_decimal(self.offline_commission_rate),
            format_decimal(self.retain_rate),
            self.remark,
        )


def format_decimal(value: float) -> str:
    return f"{value:.4f}".rstrip("0").rstrip(".")


def parse_db_config(config_path: Path) -> Dict[str, str]:
    match = DB_LINK_PATTERN.search(config_path.read_text(encoding="utf-8"))
    if not match:
        raise ValueError(f"未能从配置文件解析数据库连接: {config_path}")
    return match.groupdict()


def normalize_platform_type(value: str) -> str:
    value = value.strip().lower()
    if value in {"抖音", "douyin"}:
        return "douyin"
    if value in {"快手", "kuaishou", "kuaishow"}:
        return "kuaishou"
    if value in {"视频号", "shipinhao", "wechat", "wechatshop"}:
        return "shipinhao"
    if value in {"小红书", "xiaohongshu", "xhs"}:
        return "xiaohongshu"
    return value


def parse_platform_shop_label(label: str) -> Tuple[str, str]:
    label = label.strip()
    if not label:
        return "", ""
    for sep in ("-", "－", "—", "_", "/"):
        parts = label.split(sep, 1)
        if len(parts) == 2:
            return normalize_platform_type(parts[0]), parts[1].strip()
    normalized = normalize_platform_type(label)
    if normalized in {"douyin", "kuaishou", "shipinhao", "xiaohongshu"}:
        return normalized, ""
    return "", label


def cell_to_text(cell: xlrd.sheet.Cell) -> str:
    if cell.ctype in (xlrd.XL_CELL_EMPTY, xlrd.XL_CELL_BLANK):
        return ""
    if cell.ctype == xlrd.XL_CELL_NUMBER:
        value = float(cell.value)
        if value.is_integer():
            return str(int(value))
        return str(value).rstrip("0").rstrip(".")
    return str(cell.value).strip()


def cell_to_float(cell: xlrd.sheet.Cell) -> float:
    text = cell_to_text(cell)
    if not text:
        return 0.0
    try:
        return float(text)
    except ValueError:
        return 0.0


def build_talent_name(uid: str, account_name: str, show_id: str) -> str:
    return account_name or show_id or uid


def parse_workbook(file_path: Path) -> List[ExternalTalentRow]:
    book = xlrd.open_workbook(str(file_path))
    rows: List[ExternalTalentRow] = []

    for sheet_name in book.sheet_names():
        sheet = book.sheet_by_name(sheet_name)
        current_team = ""
        current_platform_shop = ""

        for row_idx in range(1, sheet.nrows):
            cells = [sheet.cell(row_idx, col_idx) for col_idx in range(sheet.ncols)]
            texts = [cell_to_text(cell) for cell in cells]
            if not any(texts):
                continue

            if texts[0]:
                current_team = texts[0]
            if texts[1]:
                current_platform_shop = texts[1]

            platform_type, shop_name = parse_platform_shop_label(current_platform_shop)
            row = ExternalTalentRow(
                team_name=current_team,
                platform_type=platform_type,
                shop_name=shop_name,
                platform_shop_label=current_platform_shop,
                uid=texts[2],
                account_name=texts[3],
                show_id=texts[4],
                link_code=texts[5],
                talent_name=build_talent_name(texts[2], texts[3], texts[4]),
                product_id=texts[6],
                goods_code=texts[7],
                customer_price=cell_to_float(cells[8]),
                online_commission_rate=cell_to_float(cells[9]),
                offline_commission_rate=cell_to_float(cells[10]),
                retain_rate=cell_to_float(cells[11]),
                remark=texts[12] if len(texts) > 12 else "",
                status=1,
                source_file=file_path.name,
                source_sheet=sheet_name,
                source_row_no=row_idx + 1,
            )

            if not row.team_name and not row.platform_shop_label:
                continue

            rows.append(row)

    return rows


def load_existing_rows(cursor) -> Dict[Tuple[str, ...], int]:
    cursor.execute(
        """
        SELECT id, team_name, platform_type, shop_name, uid, account_name, show_id, link_code, product_id, goods_code
        FROM external_talent_registry
        """
    )
    existing: Dict[Tuple[str, ...], int] = {}
    for row in cursor.fetchall():
        key = (
            row[1] or "",
            row[2] or "",
            row[3] or "",
            row[4] or "",
            row[5] or "",
            row[6] or "",
            row[7] or "",
            row[8] or "",
            row[9] or "",
        )
        existing.setdefault(key, row[0])
    return existing


def unique_rows(rows: Iterable[ExternalTalentRow]) -> Tuple[List[ExternalTalentRow], int]:
    seen = set()
    result: List[ExternalTalentRow] = []
    skipped = 0
    for row in rows:
        key = row.exact_row_key()
        if key in seen:
            skipped += 1
            continue
        seen.add(key)
        result.append(row)
    return result, skipped


def save_rows(rows: List[ExternalTalentRow], db_config: Dict[str, str]) -> Dict[str, int]:
    connection = pymysql.connect(
        host=db_config["host"],
        port=int(db_config["port"]),
        user=db_config["user"],
        password=db_config["password"],
        database=db_config["database"],
        charset="utf8mb4",
        autocommit=False,
    )

    inserted = 0
    updated = 0
    skipped = 0

    try:
        with connection.cursor() as cursor:
            existing = load_existing_rows(cursor)
            for row in rows:
                key = row.dedupe_key()
                payload = (
                    row.team_name,
                    row.platform_type,
                    row.shop_name,
                    row.platform_shop_label,
                    row.uid,
                    row.account_name,
                    row.show_id,
                    row.link_code,
                    row.talent_name,
                    "external",
                    row.product_id,
                    row.goods_code,
                    row.customer_price,
                    row.online_commission_rate,
                    row.offline_commission_rate,
                    row.retain_rate,
                    row.remark,
                    row.status,
                    row.source_file,
                    row.source_sheet,
                    row.source_row_no,
                )

                if key in existing:
                    cursor.execute(
                        """
                        UPDATE external_talent_registry
                        SET
                            team_name = %s,
                            platform_type = %s,
                            shop_name = %s,
                            platform_shop_label = %s,
                            uid = %s,
                            account_name = %s,
                            show_id = %s,
                            link_code = %s,
                            talent_name = %s,
                            talent_type = %s,
                            product_id = %s,
                            goods_code = %s,
                            customer_price = %s,
                            online_commission_rate = %s,
                            offline_commission_rate = %s,
                            retain_rate = %s,
                            remark = %s,
                            status = %s,
                            source_file = %s,
                            source_sheet = %s,
                            source_row_no = %s
                        WHERE id = %s
                        """,
                        payload + (existing[key],),
                    )
                    updated += 1
                    continue

                cursor.execute(
                    """
                    INSERT INTO external_talent_registry (
                        team_name, platform_type, shop_name, platform_shop_label, uid, account_name, show_id,
                        link_code, talent_name, talent_type, product_id, goods_code, customer_price,
                        online_commission_rate, offline_commission_rate, retain_rate, remark, status,
                        source_file, source_sheet, source_row_no
                    ) VALUES (
                        %s, %s, %s, %s, %s, %s, %s,
                        %s, %s, %s, %s, %s, %s,
                        %s, %s, %s, %s, %s,
                        %s, %s, %s
                    )
                    """,
                    payload,
                )
                existing[key] = cursor.lastrowid
                inserted += 1

        connection.commit()
    except Exception:
        connection.rollback()
        raise
    finally:
        connection.close()

    return {"inserted": inserted, "updated": updated, "skipped": skipped}


def main() -> None:
    parser = argparse.ArgumentParser(description="导入外部账号 Excel 到 external_talent_registry")
    parser.add_argument("excel_file", help="Excel 模板文件路径")
    parser.add_argument(
        "--config",
        default="config.yaml",
        help="数据库配置文件路径，默认读取项目根目录 config.yaml",
    )
    args = parser.parse_args()

    excel_path = Path(args.excel_file).expanduser().resolve()
    config_path = Path(args.config).expanduser().resolve()

    rows = parse_workbook(excel_path)
    unique_import_rows, duplicate_count = unique_rows(rows)
    db_config = parse_db_config(config_path)
    stats = save_rows(unique_import_rows, db_config)

    print(f"source_rows={len(rows)}")
    print(f"exact_duplicate_rows_skipped={duplicate_count}")
    print(f"import_rows={len(unique_import_rows)}")
    print(f"inserted={stats['inserted']}")
    print(f"updated={stats['updated']}")


if __name__ == "__main__":
    main()
