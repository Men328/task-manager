"""Sinh khung báo cáo ĐATN (.docx) cho hệ thống Task Manager.

Chạy:  python build_khung.py <output.docx>
Cần:   pip install python-docx

Mỗi mục gồm: tiêu đề (đã đánh số theo mẫu) + 1 ô ghi chú vàng "KHUNG" liệt kê
nội dung cần trình bày, nguồn tài liệu để đối chiếu và hình/bảng dự kiến.
Khi viết nội dung thật, xoá ô ghi chú của mục đó.
"""

import copy
import sys
from pathlib import Path

from docx import Document
from docx.enum.section import WD_SECTION
from docx.enum.table import WD_TABLE_ALIGNMENT
from docx.enum.text import WD_ALIGN_PARAGRAPH, WD_BREAK
from docx.oxml import OxmlElement
from docx.oxml.ns import qn
from docx.shared import Cm, Pt, RGBColor

FONT = "Times New Roman"

D01 = "docs/system/01-component-graph.md"
D02 = "docs/system/02-architecture.md"
D03 = "docs/system/03-business.md"
D04 = "docs/system/04-user-stories.md"
D05 = "docs/system/05-service-calls-and-data.md"
XLSX = "docs/thuyet-trinh/cong-nghe-do-an.xlsx"
README = "README.md (gốc repo)"

MAU_DIR = Path(__file__).resolve().parent / "mau"
MAU_BIA = MAU_DIR / "mau-bia.docx"          # BM.ĐT.19.06 MẪU BÌA
MAU_CAM_DOAN = MAU_DIR / "mau-cam-doan.docx"  # BM.ĐT.19.06 MẪU CAM ĐOAN

# Lấy chính file mẫu bìa làm nền để bìa giữ nguyên 100% (bảng, cột gáy dọc, logo).
doc = Document(str(MAU_BIA))


# ---------------------------------------------------------------- styles

def set_font(run_or_style, size=None, bold=None, italic=None, color=None):
    font = run_or_style.font
    font.name = FONT
    if size:
        font.size = Pt(size)
    if bold is not None:
        font.bold = bold
    if italic is not None:
        font.italic = italic
    if color:
        font.color.rgb = RGBColor.from_string(color)
    rpr = run_or_style.element.get_or_add_rPr()
    rfonts = rpr.get_or_add_rFonts()
    # Font/màu theo theme của file mẫu được Word ưu tiên hơn font chỉ định -> xoá đi.
    for attr in ("w:asciiTheme", "w:hAnsiTheme", "w:eastAsiaTheme", "w:cstheme"):
        rfonts.attrib.pop(qn(attr), None)
    col = rpr.find(qn("w:color"))
    if color and col is not None:
        for attr in ("w:themeColor", "w:themeShade", "w:themeTint"):
            col.attrib.pop(qn(attr), None)
    for attr in ("w:ascii", "w:hAnsi", "w:eastAsia", "w:cs"):
        rfonts.set(qn(attr), FONT)


def setup_styles():
    styles = doc.styles
    from docx.enum.style import WD_STYLE_TYPE

    # KHÔNG sửa style Normal: style của bìa mẫu kế thừa từ Normal.
    normal = styles.add_style("Noi dung", WD_STYLE_TYPE.PARAGRAPH)
    normal.base_style = styles["Normal"]
    set_font(normal, 13)
    pf = normal.paragraph_format
    pf.line_spacing = 1.5
    pf.space_before = Pt(0)
    pf.space_after = Pt(6)
    pf.alignment = WD_ALIGN_PARAGRAPH.JUSTIFY

    h1 = styles["Heading 1"]
    set_font(h1, 14, bold=True, color="000000")
    h1.paragraph_format.alignment = WD_ALIGN_PARAGRAPH.CENTER
    h1.paragraph_format.space_before = Pt(0)
    h1.paragraph_format.line_spacing = 1.5
    h1.paragraph_format.space_after = Pt(12)
    h1.paragraph_format.page_break_before = True
    h1.paragraph_format.keep_with_next = True

    h2 = styles["Heading 2"]
    set_font(h2, 13, bold=True, color="000000")
    h2.paragraph_format.space_before = Pt(12)
    h2.paragraph_format.line_spacing = 1.5
    h2.paragraph_format.space_after = Pt(6)
    h2.paragraph_format.alignment = WD_ALIGN_PARAGRAPH.LEFT

    h3 = styles["Heading 3"]
    set_font(h3, 13, bold=True, italic=True, color="000000")
    h3.paragraph_format.space_before = Pt(6)
    h3.paragraph_format.line_spacing = 1.5
    h3.paragraph_format.space_after = Pt(6)
    h3.paragraph_format.alignment = WD_ALIGN_PARAGRAPH.LEFT

    for name in ("Chu thich hinh", "Chu thich bang"):
        st = styles.add_style(name, WD_STYLE_TYPE.PARAGRAPH)
        st.base_style = normal
        set_font(st, 12, italic=True)
        st.paragraph_format.alignment = WD_ALIGN_PARAGRAPH.CENTER
        st.paragraph_format.space_before = Pt(3)
        st.paragraph_format.space_after = Pt(9)

    # Tiêu đề phần đầu/cuối không đánh số (vẫn vào Mục lục)
    st = styles.add_style("Tieu de phu", WD_STYLE_TYPE.PARAGRAPH)
    st.base_style = h1
    set_font(st, 14, bold=True, color="000000")

    st = styles.add_style("Ghi chu khung", WD_STYLE_TYPE.PARAGRAPH)
    st.base_style = normal
    set_font(st, 11, color="5D4037")
    st.paragraph_format.line_spacing = 1.15
    st.paragraph_format.space_after = Pt(2)
    st.paragraph_format.alignment = WD_ALIGN_PARAGRAPH.LEFT

    # "Tieu de phu" kế thừa outline level 0 của Heading 1 nên vẫn vào Mục lục.


def setup_page(section):
    section.page_height = Cm(29.7)
    section.page_width = Cm(21.0)
    section.top_margin = Cm(2.0)
    section.bottom_margin = Cm(2.0)
    section.left_margin = Cm(3.0)
    section.right_margin = Cm(2.0)


# ---------------------------------------------------------------- fields

def add_field(paragraph, instr, placeholder=""):
    run = paragraph.add_run()
    begin = OxmlElement("w:fldChar")
    begin.set(qn("w:fldCharType"), "begin")
    begin.set(qn("w:dirty"), "true")
    run._r.append(begin)

    run = paragraph.add_run()
    instr_el = OxmlElement("w:instrText")
    instr_el.set(qn("xml:space"), "preserve")
    instr_el.text = f" {instr} "
    run._r.append(instr_el)

    run = paragraph.add_run()
    sep = OxmlElement("w:fldChar")
    sep.set(qn("w:fldCharType"), "separate")
    run._r.append(sep)

    run = paragraph.add_run(placeholder)
    set_font(run, italic=True, color="808080")

    run = paragraph.add_run()
    end = OxmlElement("w:fldChar")
    end.set(qn("w:fldCharType"), "end")
    run._r.append(end)


def page_number_footer(section, fmt, restart=True):
    section.footer.is_linked_to_previous = False
    footer = section.footer
    p = footer.paragraphs[0] if footer.paragraphs else footer.add_paragraph()
    p.text = ""
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    add_field(p, "PAGE", "1")
    sect_pr = section._sectPr
    pg = sect_pr.find(qn("w:pgNumType"))
    if pg is None:
        pg = OxmlElement("w:pgNumType")
        anchor = None
        for tag in ("w:cols", "w:formProt", "w:vAlign", "w:noEndnote", "w:titlePg",
                    "w:textDirection", "w:bidi", "w:rtlGutter", "w:docGrid", "w:printerSettings"):
            anchor = sect_pr.find(qn(tag))
            if anchor is not None:
                break
        if anchor is not None:
            anchor.addprevious(pg)
        else:
            sect_pr.append(pg)
    pg.set(qn("w:fmt"), fmt)
    if restart:
        pg.set(qn("w:start"), "1")
    else:
        pg.attrib.pop(qn("w:start"), None)


def enable_update_fields_on_open():
    settings = doc.settings.element
    upd = OxmlElement("w:updateFields")
    upd.set(qn("w:val"), "true")
    for tag in ("w:hdrShapeDefaults", "w:footnotePr", "w:endnotePr", "w:compat", "w:docVars",
                "w:rsids", "m:mathPr", "w:attachedSchema", "w:themeFontLang", "w:clrSchemeMapping",
                "w:doNotIncludeSubdocsInStats", "w:doNotAutoCompressPictures", "w:forceUpgrade",
                "w:captions", "w:readModeInkLockDown", "w:smartTagType", "sl:schemaLibrary",
                "w:shapeDefaults", "w:doNotEmbedSmartTags", "w:decimalSymbol", "w:listSeparator"):
        prefix = tag.split(":")[0]
        if prefix not in ("w",):
            continue
        anchor = settings.find(qn(tag))
        if anchor is not None:
            anchor.addprevious(upd)
            return
    settings.append(upd)


# ---------------------------------------------------------------- builders

def para(text="", style="Noi dung", align=None, size=None, bold=None, italic=None, space_after=None):
    p = doc.add_paragraph(style=style)
    if text:
        r = p.add_run(text)
        set_font(r, size, bold, italic)
    if align is not None:
        p.alignment = align
    if space_after is not None:
        p.paragraph_format.space_after = Pt(space_after)
    return p


def h1(text):
    return doc.add_heading(text, level=1)


def front_title(text):
    p = doc.add_paragraph(text, style="Tieu de phu")
    return p


def h2(text):
    return doc.add_heading(text, level=2)


def h3(text):
    return doc.add_heading(text, level=3)


def shade(cell, hex_fill):
    tc_pr = cell._tc.get_or_add_tcPr()
    shd = OxmlElement("w:shd")
    shd.set(qn("w:val"), "clear")
    shd.set(qn("w:color"), "auto")
    shd.set(qn("w:fill"), hex_fill)
    tc_pr.append(shd)


def cell_borders(cell, color="F9A825"):
    tc_pr = cell._tc.get_or_add_tcPr()
    borders = OxmlElement("w:tcBorders")
    for side in ("top", "left", "bottom", "right"):
        b = OxmlElement(f"w:{side}")
        b.set(qn("w:val"), "single")
        b.set(qn("w:sz"), "8")
        b.set(qn("w:color"), color)
        borders.append(b)
    tc_pr.append(borders)


def box(can=None, nguon=None, hinh=None, bang=None, luu_y=None):
    """Ô ghi chú KHUNG: xoá khi viết nội dung thật."""
    table = doc.add_table(rows=1, cols=1)
    table.alignment = WD_TABLE_ALIGNMENT.CENTER
    cell = table.rows[0].cells[0]
    cell_borders(cell)
    shade(cell, "FFF8E1")
    cell.paragraphs[0].text = ""

    first = True

    def line(label, items):
        nonlocal first
        if not items:
            return
        p = cell.paragraphs[0] if first else cell.add_paragraph()
        first = False
        p.style = doc.styles["Ghi chu khung"]
        r = p.add_run(label)
        set_font(r, 11, bold=True, color="E65100")
        for item in items:
            q = cell.add_paragraph(style="Ghi chu khung")
            q.paragraph_format.left_indent = Cm(0.4)
            r = q.add_run("• " + item)
            set_font(r, 11, color="5D4037")

    line("[KHUNG] Cần trình bày:", can)
    line("Nguồn đối chiếu:", nguon)
    line("Hình dự kiến:", hinh)
    line("Bảng dự kiến:", bang)
    line("Lưu ý:", luu_y)
    doc.add_paragraph().paragraph_format.space_after = Pt(0)


def figure(caption):
    p = para("[Chèn hình]", align=WD_ALIGN_PARAGRAPH.CENTER, italic=True)
    p.runs[0].font.color.rgb = RGBColor(0x9E, 0x9E, 0x9E)
    doc.add_paragraph(caption, style="Chu thich hinh")


def table(caption, headers, rows=1):
    if caption:
        doc.add_paragraph(caption, style="Chu thich bang")
    t = doc.add_table(rows=1 + rows, cols=len(headers))
    t.style = "Table Grid"
    t.alignment = WD_TABLE_ALIGNMENT.CENTER
    for i, h in enumerate(headers):
        c = t.rows[0].cells[i]
        c.text = ""
        r = c.paragraphs[0].add_run(h)
        set_font(r, 12, bold=True)
        c.paragraphs[0].alignment = WD_ALIGN_PARAGRAPH.CENTER
        shade(c, "E3F2FD")
    for row in t.rows[1:]:
        for c in row.cells:
            c.text = ""
            r = c.paragraphs[0].add_run("…")
            set_font(r, 12)
    fit_table(t, [headers] + [["…"] * len(headers)] * rows)
    doc.add_paragraph().paragraph_format.space_after = Pt(0)
    return t



TABLE_FONT = 11          # cỡ chữ trong bảng
TEXT_WIDTH_CM = 16.0     # A4 21 cm - lề trái 3 cm - lề phải 2 cm
CHAR_CM = 0.2            # bề rộng trung bình 1 ký tự Times New Roman 11pt
NARROW_HEADERS = {"STT", "Mã", "#"}


def _plain(text):
    return str(text).replace("**", "").replace("*", "").replace("`", "")


KEEP_WHOLE_MAX_ROWS = 10  # bảng ngắn hơn thì giữ trọn trên 1 trang


def keep_table_together(t):
    """Chú thích + tiêu đề cột không bị bỏ lại cuối trang; ưu tiên đẩy cả bảng xuống trang mới."""
    rows = t.rows
    whole = len(rows) <= KEEP_WHOLE_MAX_ROWS
    for ri, row in enumerate(rows):
        tr_pr = row._tr.get_or_add_trPr()
        if tr_pr.find(qn("w:cantSplit")) is None:
            tr_pr.append(OxmlElement("w:cantSplit"))      # không cắt đôi 1 dòng
        if ri == 0 and tr_pr.find(qn("w:tblHeader")) is None:
            tr_pr.append(OxmlElement("w:tblHeader"))      # lặp tiêu đề cột ở đầu mỗi trang
        last = ri == len(rows) - 1
        keep = (ri == 0) or (whole and not last)
        for cell in row.cells:
            for p in cell.paragraphs:
                p.paragraph_format.keep_with_next = keep
    prev = t._tbl.getprevious()
    if prev is not None and prev.tag == qn("w:p"):
        from docx.text.paragraph import Paragraph as _Pp
        _Pp(prev, doc).paragraph_format.keep_with_next = True   # chú thích dính với bảng


def fit_table(t, rows):
    """Chia bề rộng cột theo nội dung, cố định bố cục để chữ không bị vỡ từng âm tiết."""
    ncols = len(rows[0])
    mins, weights = [], []
    for ci in range(ncols):
        col = [_plain(r[ci]) if ci < len(r) else "" for r in rows]
        header, body = col[0], col[1:] or [""]
        longest_word = max((len(w) for v in col for w in v.split()), default=3)
        avg = sum(len(v) for v in body) / len(body)
        # từ dài nhất phải vừa 1 dòng (mã lỗi rất dài thì cho ngắt), và ô trung bình không quá ~3 dòng
        min_cm = max(min(longest_word * CHAR_CM + 0.35, 3.0), min(avg * CHAR_CM / 3 + 0.35, 3.2))
        if header in NARROW_HEADERS:
            min_cm = max(len(header), max(len(v) for v in body)) * CHAR_CM + 0.4
        mins.append(min_cm)
        weights.append(0 if header in NARROW_HEADERS else max(avg, 4) ** 0.9)
    spare = TEXT_WIDTH_CM - sum(mins)
    if spare < 0:
        mins = [m * TEXT_WIDTH_CM / sum(mins) for m in mins]
        spare = 0
    total_w = sum(weights) or 1
    widths = [m + spare * w / total_w for m, w in zip(mins, weights)]

    t.autofit = False
    tbl_pr = t._tbl.tblPr
    tbl_w = tbl_pr.find(qn("w:tblW"))
    if tbl_w is None:
        tbl_w = OxmlElement("w:tblW")
        tbl_pr.append(tbl_w)
    tbl_w.set(qn("w:w"), str(int(TEXT_WIDTH_CM * 567)))
    tbl_w.set(qn("w:type"), "dxa")
    if tbl_pr.find(qn("w:tblCellMar")) is None:
        mar = OxmlElement("w:tblCellMar")
        for side, val in (("top", 30), ("left", 85), ("bottom", 30), ("right", 85)):
            e = OxmlElement(f"w:{side}")
            e.set(qn("w:w"), str(val))
            e.set(qn("w:type"), "dxa")
            mar.append(e)
        look = tbl_pr.find(qn("w:tblLook"))
        if look is not None:
            look.addprevious(mar)
        else:
            tbl_pr.append(mar)
    for ci, w in enumerate(widths):
        t.columns[ci].width = Cm(w)
        for cell in t.columns[ci].cells:
            cell.width = Cm(w)
    for row in t.rows:
        for cell in row.cells:
            for p in cell.paragraphs:
                pf = p.paragraph_format
                pf.space_before = Pt(0)
                pf.space_after = Pt(0)
                pf.line_spacing = 1.15
                pf.first_line_indent = Cm(0)
                for r in p.runs:
                    r.font.size = Pt(TABLE_FONT)
    keep_table_together(t)


# ---------------------------------------------------------------- nội dung chương từ file .md
NOI_DUNG_DIR = Path(__file__).resolve().parent.parent / "noi-dung"
HINH_DIR = Path(__file__).resolve().parent.parent / "hinh"


def add_rich_text(p, text, size=None):
    """Hỗ trợ **đậm** và *nghiêng* đơn giản trong một đoạn."""
    import re as _re
    for part in _re.split(r"(`[^`]+`|\*\*[^*]+\*\*|\*[^*]+\*)", text):
        if not part:
            continue
        if part.startswith("`"):
            # Tên kỹ thuật: giữ Times New Roman; chèn khoảng trắng độ rộng 0 sau / _ . để
            # đường dẫn dài ngắt dòng được, tránh dãn chữ khi căn đều hai bên.
            code = _re.sub(r"([/_.])", "\\1\u200b", part[1:-1])
            r = p.add_run(code)
            set_font(r, size)
        elif part.startswith("**"):
            # Chỉ in đậm nhãn ở đầu đoạn / đầu gạch đầu dòng; giữa câu thì để chữ thường.
            lead = not "".join(x.text for x in p.runs).strip(" •–\t")
            r = p.add_run(part[2:-2])
            set_font(r, size, bold=lead)
        elif part.startswith("*"):
            r = p.add_run(part[1:-1])
            set_font(r, size, italic=True)
        else:
            r = p.add_run(part)
            set_font(r, size)


def content_table(caption, rows):
    if caption:
        doc.add_paragraph(caption, style="Chu thich bang")
    header, body_rows = rows[0], rows[1:]
    t = doc.add_table(rows=len(rows), cols=len(header))
    t.style = "Table Grid"
    t.alignment = WD_TABLE_ALIGNMENT.CENTER
    for ci, h in enumerate(header):
        c = t.rows[0].cells[ci]
        c.text = ""
        r = c.paragraphs[0].add_run(h)
        set_font(r, 12, bold=True)
        c.paragraphs[0].alignment = WD_ALIGN_PARAGRAPH.CENTER
        shade(c, "E3F2FD")
    for ri, row in enumerate(body_rows, start=1):
        for ci, val in enumerate(row):
            c = t.rows[ri].cells[ci]
            c.text = ""
            add_rich_text(c.paragraphs[0], val, size=12)
            c.paragraphs[0].alignment = WD_ALIGN_PARAGRAPH.LEFT
    fit_table(t, rows)
    doc.add_paragraph().paragraph_format.space_after = Pt(0)


def content_figure(img, caption):
    path = HINH_DIR / img
    if path.exists():
        p = doc.add_paragraph(style="Noi dung")
        p.alignment = WD_ALIGN_PARAGRAPH.CENTER
        p.paragraph_format.first_line_indent = Cm(0)
        p.paragraph_format.keep_with_next = True
        import struct as _struct
        with open(path, "rb") as fh:  # đọc kích thước từ header PNG (IHDR)
            w_px, h_px = _struct.unpack(">II", fh.read(24)[16:24])
        max_w, max_h = 15.5, 21.0
        width = min(max_w, w_px / 96 * 2.54 / 2)  # ảnh render ở scale 2
        if width * h_px / w_px > max_h:
            width = max_h * w_px / h_px
        p.add_run().add_picture(str(path), width=Cm(width))
    else:
        figure_placeholder = para(f"[Chèn hình: {img}]", align=WD_ALIGN_PARAGRAPH.CENTER, italic=True)
        figure_placeholder.runs[0].font.color.rgb = RGBColor(0x9E, 0x9E, 0x9E)
    doc.add_paragraph(caption, style="Chu thich hinh")


def render_chapter(md_name, first=False):
    lines = (NOI_DUNG_DIR / md_name).read_text(encoding="utf-8").splitlines()
    i = 0
    while i < len(lines):
        line = lines[i].rstrip()
        i += 1
        if not line.strip() or line.startswith("<!--"):
            continue
        if line.startswith("# "):
            hp = h1(line[2:].strip())
            if first:
                hp.paragraph_format.page_break_before = False
        elif line.startswith("### "):
            h3(line[4:].strip())
        elif line.startswith("## "):
            h2(line[3:].strip())
        elif line.startswith("[HINH:"):
            img, caption = [x.strip() for x in line[len("[HINH:"):-1].split("|", 1)]
            content_figure(img, caption)
        elif line.startswith("[BANG:"):
            caption = line[len("[BANG:"):-1].strip()
            rows = []
            while i < len(lines) and lines[i].strip().startswith("|"):
                cells = [c.strip() for c in lines[i].strip().strip("|").split("|")]
                if not all(set(c) <= set("-: ") for c in cells):
                    rows.append(cells)
                i += 1
            content_table(caption, rows)
        elif line.lstrip().startswith(("- ", "+ ")):
            level = 1 if line.startswith(("  ", "\t")) else 0
            p = doc.add_paragraph(style="Noi dung")
            p.paragraph_format.left_indent = Cm(0.75 + 0.75 * level)
            p.paragraph_format.first_line_indent = Cm(-0.5)
            p.paragraph_format.space_after = Pt(3)
            marker = "–" if level else "•"
            add_rich_text(p, f"{marker}  {line.lstrip()[2:].strip()}")
        else:
            p = doc.add_paragraph(style="Noi dung")
            p.paragraph_format.first_line_indent = Cm(1)
            add_rich_text(p, line.strip())


def page_break():
    doc.add_paragraph().add_run().add_break(WD_BREAK.PAGE)


TEST_COLS = ["STT", "Chức năng", "Dữ liệu đầu vào", "Kết quả mong đợi", "Kết quả thực tế", "Trạng thái"]
TABLE_COLS = ["STT", "Tên cột", "Kiểu dữ liệu", "Ràng buộc", "Mô tả"]


# ================================================================= CONTENT

setup_styles()
enable_update_fields_on_open()

# ---------------------------------------------------------------- Thông tin sinh viên
SV_HO_TEN = "Trần Thị Mến"
SV_MSSV = "22010328"
SV_KHOA = "16"
SV_NGANH = "Công nghệ thông tin"
SV_HE = "Chính quy"
GVHD = "ThS. Vũ Thị Ngọc Anh"
NAM = "2026"
DE_TAI = "Hệ thống quản lí công việc, lịch cá nhân"

# ---------------------------------------------------------------- Bìa (từ file mẫu)
# Chỉ điền giá trị đã chắc chắn; trường cá nhân giữ nguyên chỗ trống "(Ghi ...)" của mẫu.
SPINE_OLD = "HỌ VÀ TÊN SINH VIÊN                                                              NGÀNH"
_spine_gap = len(SPINE_OLD) - len(SV_HO_TEN) - len(SV_NGANH)
COVER_FILL = {
    "ĐỐ ÁN/KHÓA LUẬN TỐT NGHIỆP": "ĐỒ ÁN TỐT NGHIỆP",
    "(Chữ in hoa thẳng, in đậm cỡ 20-24)": "",
    "(Chữ in hoa thẳng, in đậm cỡ 16)": "",
    # Gáy bìa: giữ nguyên tổng độ dài dòng để chữ vẫn nằm vừa gáy.
    SPINE_OLD.strip(): SV_HO_TEN.upper() + " " * _spine_gap + SV_NGANH.upper(),
    "Sinh viên: (Ghi Họ và tên sinh viên)": f"Sinh viên: {SV_HO_TEN}",
    "Mã số sinh viên: (Ghi mã sinh viên)": f"Mã số sinh viên: {SV_MSSV}",
    "Khóa: (Ghi khóa nhập học)": f"Khóa: {SV_KHOA}",
    "Ngành: (Ghi ngành học)": f"Ngành: {SV_NGANH}",
    "Hệ: (Ghi hệ nhập học)": f"Hệ: {SV_HE}",
    "Giảng viên hướng dẫn: (Ghi học hàm học vị. Họ và tên GVHD)": f"Giảng viên hướng dẫn: {GVHD}",
    "Hà Nội -  Năm": f"Hà Nội - {NAM}",
    "TÊN ĐỀ TÀI": DE_TAI.upper(),
    "TRANG BÌA PHỤ": "",  # nhãn hướng dẫn giữa bìa chính và bìa phụ, không in
}


def all_paragraphs(element):
    from docx.text.paragraph import Paragraph
    return [Paragraph(p, doc) for p in element.iter(qn("w:p"))]


def set_paragraph_text(p, text, keep_tail=False):
    runs = p.runs
    runs[0].text = text
    if keep_tail:
        return
    for r in runs[1:]:
        r._r.getparent().remove(r._r)


for p in all_paragraphs(doc.element.body):
    key = p.text.strip()
    if key in COVER_FILL and p.runs:
        set_paragraph_text(p, COVER_FILL[key])

# ---- Giữ mỗi bìa gọn trong 1 trang
# Mẫu đặt chiều cao dòng "tối thiểu" 315 twip nhưng chữ giãn dòng 1.5 -> Word nới dòng
# và bảng tràn sang trang sau. Sửa: dòng trống cao cố định đúng số đo của mẫu, dòng có
# chữ giãn dòng đơn, bỏ đoạn trống thừa, ngắt trang rõ ràng giữa bìa chính và bìa phụ.
from docx.text.paragraph import Paragraph as _CoverP  # noqa: E402
from docx.enum.text import WD_LINE_SPACING  # noqa: E402


def tiny_paragraph(p_el, page_break=False):
    para_ = _CoverP(p_el, doc)
    for r in list(para_.runs):
        r._r.getparent().remove(r._r)
    pf_ = para_.paragraph_format
    pf_.space_before = Pt(0)
    pf_.space_after = Pt(0)
    pf_.line_spacing_rule = WD_LINE_SPACING.EXACTLY
    pf_.line_spacing = Pt(1)
    pf_.page_break_before = page_break
    pf_.keep_with_next = False
    pf_.keep_together = False


body = doc.element.body
cover_tables = body.findall(qn("w:tbl"))
for tbl in cover_tables:
    for tr in tbl.findall(qn("w:tr")):
        cells = tr.findall(qn("w:tc"))
        content_cells = cells[1:]  # cột 0 là gáy bìa (chữ dọc, gộp dọc toàn bảng)
        has_text = any((t.text or "").strip() for c in content_cells for t in c.iter(qn("w:t"))) or any(
            next(c.iter(qn("w:drawing"), qn("w:pict")), None) is not None for c in content_cells)
        trh = tr.find(qn("w:trPr") + "/" + qn("w:trHeight"))
        if trh is not None:
            trh.set(qn("w:hRule"), "atLeast" if has_text else "exact")
        for c in content_cells:
            for p_el in c.iter(qn("w:p")):
                pf_ = _CoverP(p_el, doc).paragraph_format
                pf_.space_before = Pt(0)
                pf_.space_after = Pt(0)
                pf_.line_spacing_rule = WD_LINE_SPACING.SINGLE

# Đoạn trống trước bìa chính: bỏ. Đoạn giữa 2 bìa: chỉ còn là ngắt trang.
# Đoạn trống sau bìa phụ (space-after 760 twip): bỏ.
children = list(body)
first_tbl = children.index(cover_tables[0])
for el in children[:first_tbl]:
    if el.tag == qn("w:p"):
        body.remove(el)
children = list(body)
i0, i1 = children.index(cover_tables[0]), children.index(cover_tables[1])
between = [el for el in children[i0 + 1:i1] if el.tag == qn("w:p")]
for el in between[1:]:
    body.remove(el)
tiny_paragraph(between[0], page_break=True)
for el in list(body)[list(body).index(cover_tables[1]) + 1:]:
    if el.tag == qn("w:p"):
        body.remove(el)

for tbl in cover_tables:
    for tr in tbl.findall(qn("w:tr")):
        tcs = tr.findall(qn("w:tc"))
        for k, tc in enumerate(tcs[:-1]):
            if "".join(t.text or "" for t in tc.iter(qn("w:t"))).startswith("Ngành:"):
                nxt = tcs[k + 1]
                if "".join(t.text or "" for t in nxt.iter(qn("w:t"))).strip():
                    break
                pr, npr = tc.get_or_add_tcPr(), nxt.get_or_add_tcPr()
                span = pr.find(qn("w:gridSpan"))
                span.set(qn("w:val"), str(int(span.get(qn("w:val"))) + 1))
                w, nw = pr.find(qn("w:tcW")), npr.find(qn("w:tcW"))
                w.set(qn("w:w"), str(int(w.get(qn("w:w"))) + int(nw.get(qn("w:w")))))
                tr.remove(nxt)
                break

# Tên đề tài 1 dòng: giữ cỡ 16pt của mẫu, thu khoảng cách ký tự 0,6pt
# (16pt cần 13,5 cm, ô chỉ có 12,8 cm; sau khi thu còn ~12,6 cm).
TITLE_SPACING = -12  # đơn vị 1/20 pt
for p_el in body.iter(qn("w:p")):
    if "".join(t.text or "" for t in p_el.iter(qn("w:t"))) == DE_TAI.upper():
        for r in p_el.findall(qn("w:r")):
            rpr = r.get_or_add_rPr()
            sp = rpr.find(qn("w:spacing"))
            if sp is None:
                sp = OxmlElement("w:spacing")
                anchor = None
                for tag in ("w:w", "w:kern", "w:position", "w:sz", "w:szCs", "w:highlight", "w:u",
                            "w:effect", "w:bdr", "w:shd", "w:fitText", "w:vertAlign", "w:lang"):
                    anchor = rpr.find(qn(tag))
                    if anchor is not None:
                        break
                if anchor is not None:
                    anchor.addprevious(sp)
                else:
                    rpr.append(sp)
            sp.set(qn("w:val"), str(TITLE_SPACING))

# Logo: mẫu kéo giãn ảnh 271x186 px (tỉ lệ 1,46) thành 5,58 x 2,70 cm (tỉ lệ 2,07).
# Giữ chiều cao 2,70 cm, đặt chiều rộng theo đúng tỉ lệ ảnh gốc, căn lại tâm như cũ.
WP = "{http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing}"
A = "{http://schemas.openxmlformats.org/drawingml/2006/main}"
import struct as _struct  # noqa: E402
for anchor_el in body.iter(WP + "anchor"):
    blip = next(anchor_el.iter(A + "blip"), None)
    if blip is None:
        continue
    rid = blip.get("{http://schemas.openxmlformats.org/officeDocument/2006/relationships}embed")
    png = doc.part.related_parts[rid].blob
    px_w, px_h = _struct.unpack(">II", png[16:24])
    ext = anchor_el.find(WP + "extent")
    cx_old, cy = int(ext.get("cx")), int(ext.get("cy"))
    cx_new = int(cy * px_w / px_h)
    ext.set("cx", str(cx_new))
    for a_ext in anchor_el.iter(A + "ext"):
        if a_ext.get("cx"):
            a_ext.set("cx", str(cx_new))
    off = anchor_el.find(WP + "positionH").find(WP + "posOffset")
    off.text = str(int(off.text) + (cx_old - cx_new) // 2)


# ---- Căn giữa nội dung bìa theo khung (phần bên phải gáy)
# Mẫu đặt nội dung trong ô bắt đầu sau 2 cột trống cạnh gáy nên chữ lệch phải ~1 cm so với khung.
# Gộp ô trống đó vào ô nội dung, căn logo về tâm khung, thụt lề khối thông tin cho cân giữa.
def _tc_text(tc):
    return "".join(t.text or "" for t in tc.iter(qn("w:t")))


def _span(tc):
    g = tc.get_or_add_tcPr().find(qn("w:gridSpan"))
    return int(g.get(qn("w:val"))) if g is not None else 1


def _set_span(tc, n):
    pr = tc.get_or_add_tcPr()
    g = pr.find(qn("w:gridSpan"))
    if g is None:
        g = OxmlElement("w:gridSpan")
        pr.find(qn("w:tcW")).addnext(g)
    g.set(qn("w:val"), str(n))


def _width(tc):
    return int(tc.get_or_add_tcPr().find(qn("w:tcW")).get(qn("w:w")))


def _text_width_twips(text, pt, bold):
    try:
        import unicodedata
        from fontTools.ttLib import TTFont
        name = "LiberationSerif-Bold.ttf" if bold else "LiberationSerif-Regular.ttf"
        font = TTFont("/usr/share/fonts/truetype/liberation/" + name)
        cmap, hmtx, upm = font.getBestCmap(), font["hmtx"], font["head"].unitsPerEm
        units = 0
        for ch in unicodedata.normalize("NFC", text):
            gl = cmap.get(ord(ch)) or cmap.get(ord(unicodedata.normalize("NFD", ch)[0]))
            units += hmtx[gl][0]
        return units / upm * pt * 20
    except Exception:  # không có fontTools/font: ước lượng
        return len(text) * pt * 20 * (0.55 if bold else 0.5)


CELL_PAD = 108  # lề trái/phải mặc định của ô (twip)
INFO_KEYS = ("Sinh viên:", "Mã số sinh viên:", "Ngành:", "Giảng viên hướng dẫn:")
for tbl in cover_tables:
    grid = [int(g.get(qn("w:w"))) for g in tbl.find(qn("w:tblGrid")).findall(qn("w:gridCol"))]
    spine = grid[0]
    frame_center = (spine + sum(grid)) / 2
    info_paras, second_col_texts, first_col_limits = [], [], []
    for tr in tbl.findall(qn("w:tr")):
        tcs = tr.findall(qn("w:tc"))
        if len(tcs) < 3 or _tc_text(tcs[1]).strip() or next(tcs[1].iter(qn("w:drawing")), None) is not None:
            continue
        if _span(tcs[1]) != 2:
            continue
        gap, content = tcs[1], tcs[2]
        _set_span(content, _span(content) + _span(gap))
        content.get_or_add_tcPr().find(qn("w:tcW")).set(qn("w:w"), str(_width(content) + _width(gap)))
        tr.remove(gap)
        text = _tc_text(content)
        if text.startswith(INFO_KEYS):
            info_paras.append(content.find(qn("w:p")))
            col2 = tr.findall(qn("w:tc"))[2:3]
            if col2 and _tc_text(col2[0]).strip():
                second_col_texts.append((spine + _width(content), _tc_text(col2[0])))
                first_col_limits.append((_width(content) - 2 * CELL_PAD, text))
    # khối thông tin: thụt lề để khối (từ mép trái chữ tới mép phải dài nhất) nằm giữa khung
    if info_paras:
        bold = True
        widths = [_text_width_twips("".join(t.text or "" for t in p.iter(qn("w:t"))), 13, bold) for p in info_paras]
        right_col = [x + CELL_PAD + _text_width_twips(t, 13, bold) for x, t in second_col_texts]
        left = frame_center - max(widths) / 2
        block_right = max([left + w for w in widths] + right_col)
        left = frame_center - (block_right - left) / 2
        indent = max(0, int(left - spine - CELL_PAD))
        # không để dòng ở cột trái (cạnh Khóa/Hệ) tràn sang cột phải
        for avail, txt in first_col_limits:
            indent = min(indent, int(avail - _text_width_twips(txt, 13, bold) - 85))  # chừa ~0,15 cm
        for p_el in info_paras:
            _CoverP(p_el, doc).paragraph_format.left_indent = Cm(indent / 567)
    # logo: đặt tâm logo vào tâm khung (vị trí tính theo ô chứa logo)
    for tr in tbl.findall(qn("w:tr")):
        x = 0
        for tc in tr.findall(qn("w:tc")):
            anchor_el = next(tc.iter(WP + "anchor"), None)
            if anchor_el is not None and next(anchor_el.iter(A + "blip"), None) is not None:
                cx_twip = int(anchor_el.find(WP + "extent").get("cx")) / 635
                off_twip = frame_center - x - cx_twip / 2
                anchor_el.find(WP + "positionH").find(WP + "posOffset").text = str(int(off_twip * 635))
            x += _width(tc)

# Bỏ header/footer của mẫu (footer chứa mã biểu mẫu "BM.ĐT.19.06 ...") khỏi trang bìa.
body_sect = doc.element.body.find(qn("w:sectPr"))
for tag in ("w:headerReference", "w:footerReference"):
    for ref in body_sect.findall(qn(tag)):
        body_sect.remove(ref)

# ---------------------------------------------------------------- Phần đầu (đánh số La Mã)
front = doc.add_section(WD_SECTION.NEW_PAGE)
setup_page(front)
# Phần mở đầu không đánh số trang (như báo cáo mẫu): độ dài mục lục/danh mục không làm lệch số trang các chương.
front.footer.is_linked_to_previous = False
# Đoạn mang ngắt section ngay sau bìa phụ: thu nhỏ để không đẩy sang trang trắng.
for p_el in body.findall(qn("w:p")):
    if p_el.find(qn("w:pPr") + "/" + qn("w:sectPr")) is not None:
        tiny_paragraph(p_el)
        break

# ---------------------------------------------------------------- Lời cam đoan (chép nguyên từ mẫu)
CAM_DOAN_FILL = {
    "Họ và tên:": (f"Họ và tên: {SV_HO_TEN}", False),
    "Mã sinh viên:\tLớp:\t..": (f"Mã sinh viên: {SV_MSSV}", True),  # giữ phần "Lớp:" để điền
    "Ngành:": (f"Ngành: {SV_NGANH}", False),
    "Tôi đã thực hiện đồ án/khóa luận tốt nghiệp với đề tài:":
        (f"Tôi đã thực hiện đồ án tốt nghiệp với đề tài: {DE_TAI}.", False),
    "Tôi xin cam đoan đây là đề tài nghiên cứu của riêng tôi và được sự hướng dẫn của:":
        (f"Tôi xin cam đoan đây là đề tài nghiên cứu của riêng tôi và được sự hướng dẫn của: {GVHD}.", False),
}
cam_doan = Document(str(MAU_CAM_DOAN))
text_width = int((front.page_width - front.left_margin - front.right_margin) / 635)  # EMU -> twip
sect_tag = qn("w:sectPr")
for el in cam_doan.element.body:
    if el.tag == sect_tag:
        continue
    clone = copy.deepcopy(el)
    if clone.tag == qn("w:tbl"):
        grid = clone.findall(".//" + qn("w:gridCol"))
        total = sum(int(g.get(qn("w:w"))) for g in grid)
        if total > text_width:
            ratio = text_width / total
            for g in grid:
                g.set(qn("w:w"), str(int(int(g.get(qn("w:w"))) * ratio)))
            for w in clone.iter(qn("w:tcW"), qn("w:tblW")):
                if w.get(qn("w:type")) == "dxa":
                    w.set(qn("w:w"), str(int(int(w.get(qn("w:w"))) * ratio)))
    doc.element.body.find(sect_tag).addprevious(clone)

from docx.text.paragraph import Paragraph as _P  # noqa: E402
for el in doc.element.body.iter(qn("w:p")):
    p = _P(el, doc)
    key = p.text.strip()
    if key in CAM_DOAN_FILL and p.runs:
        set_paragraph_text(p, *CAM_DOAN_FILL[key])


def intro_title(text):
    """Tiêu đề phần mở đầu: cùng style 'Heading #5' với LỜI CAM ĐOAN của mẫu."""
    p = doc.add_paragraph(style="Heading #5")
    p.alignment = WD_ALIGN_PARAGRAPH.CENTER
    p.paragraph_format.page_break_before = True
    p.add_run(text)
    return p


intro_title("LỜI CẢM ƠN")
render_chapter("loi-cam-on.md")
for text, bold in [(f"Hà Nội, ngày … tháng … năm {NAM}", False), ("Sinh viên", True), ("", False), ("", False),
                   (SV_HO_TEN, True)]:
    sp = para(text, align=WD_ALIGN_PARAGRAPH.CENTER, bold=bold, italic=(not bold and bool(text)), space_after=0)
    sp.paragraph_format.left_indent = Cm(8)

intro_title("LỜI MỞ ĐẦU")
render_chapter("loi-mo-dau.md")

intro_title("MỤC LỤC")
p = para()
add_field(p, 'TOC \\o "1-3" \\h \\z \\u',
          "Mục lục tự động — chuột phải → Update Field (hoặc F9) để cập nhật.")

intro_title("DANH MỤC HÌNH ẢNH")
p = para()
add_field(p, 'TOC \\h \\z \\t "Chu thich hinh,1"',
          "Danh mục hình tự động — chuột phải → Update Field.")

intro_title("DANH MỤC BẢNG")
p = para()
add_field(p, 'TOC \\h \\z \\t "Chu thich bang,1"',
          "Danh mục bảng tự động — chuột phải → Update Field.")

intro_title("DANH MỤC TỪ VIẾT TẮT")
content_table("", [['Từ viết tắt', 'Tiếng Anh', 'Nghĩa tiếng Việt'], ['AI', 'Artificial Intelligence', 'Trí tuệ nhân tạo'], ['API', 'Application Programming Interface', 'Giao diện lập trình ứng dụng'], ['CRUD', 'Create, Read, Update, Delete', 'Tạo, đọc, cập nhật, xoá'], ['CSDL', 'Database', 'Cơ sở dữ liệu'], ['DI', 'Dependency Injection', 'Tiêm phụ thuộc'], ['ERD', 'Entity Relationship Diagram', 'Sơ đồ thực thể liên kết'], ['gRPC', 'gRPC Remote Procedure Call', 'Khung gọi thủ tục từ xa'], ['HTTP', 'Hypertext Transfer Protocol', 'Giao thức truyền siêu văn bản'], ['JSON', 'JavaScript Object Notation', 'Định dạng dữ liệu JSON'], ['JWT', 'JSON Web Token', 'Mã thông báo web dạng JSON'], ['LLM', 'Large Language Model', 'Mô hình ngôn ngữ lớn'], ['OAuth', 'Open Authorization', 'Giao thức uỷ quyền mở'], ['OIDC', 'OpenID Connect', 'Lớp định danh trên OAuth 2.0'], ['PKCE', 'Proof Key for Code Exchange', 'Khoá chứng minh khi đổi mã uỷ quyền'], ['REST', 'Representational State Transfer', 'Phong cách kiến trúc REST'], ['RPC', 'Remote Procedure Call', 'Gọi thủ tục từ xa'], ['RRULE', 'Recurrence Rule', 'Quy tắc lặp lại (chuẩn iCalendar)'], ['S3', 'Simple Storage Service', 'Chuẩn API lưu trữ đối tượng'], ['SPA', 'Single Page Application', 'Ứng dụng web một trang'], ['SQL', 'Structured Query Language', 'Ngôn ngữ truy vấn có cấu trúc'], ['UI', 'User Interface', 'Giao diện người dùng'], ['UUID', 'Universally Unique Identifier', 'Mã định danh duy nhất toàn cục'], ['WS', 'WebSocket', 'Giao thức WebSocket']])

# ---------------------------------------------------------------- Phần chính (số Ả Rập)
main = doc.add_section(WD_SECTION.NEW_PAGE)
setup_page(main)
page_number_footer(main, "decimal")  # Chương 1 bắt đầu từ trang 1

# ================= CHƯƠNG 1, 2 (đã viết nội dung: docs/bao-cao/noi-dung/)
render_chapter("chuong-1.md", first=True)
render_chapter("chuong-2.md")

# ================= CHƯƠNG 3 (đã viết nội dung)
render_chapter("chuong-3.md")

# ================= CHƯƠNG 4, 5 (đã viết nội dung)
render_chapter("chuong-4.md")
render_chapter("chuong-5.md")

# ================= KẾT LUẬN
front_title("KẾT LUẬN")
render_chapter("ket-luan.md")

# ================= TÀI LIỆU THAM KHẢO
front_title("DANH MỤC TÀI LIỆU THAM KHẢO")
refs = [
    "S. Newman, Building Microservices: Designing Fine-Grained Systems, 2nd ed., O’Reilly Media, 2021.",
    "J. Lewis & M. Fowler, “Microservices – a definition of this new architectural term”, https://martinfowler.com/articles/microservices.html, 2014.",
    "M. Kleppmann, Designing Data-Intensive Applications, O’Reilly Media, 2017.",
    "The Go Authors, “The Go Programming Language Documentation”, https://go.dev/doc/.",
    "The gRPC Authors, “gRPC Documentation”, https://grpc.io/docs/.",
    "Google, “Protocol Buffers Documentation”, https://protobuf.dev/.",
    "gRPC-Gateway Authors, “gRPC-Gateway Documentation”, https://grpc-ecosystem.github.io/grpc-gateway/.",
    "R. T. Fielding, Architectural Styles and the Design of Network-based Software Architectures, Doctoral dissertation, University of California, Irvine, 2000.",
    "Apache Software Foundation, “Apache Kafka Documentation”, https://kafka.apache.org/documentation/.",
    "I. Fette & A. Melnikov, “The WebSocket Protocol”, RFC 6455, IETF, 2011.",
    "Soketi, “Soketi Documentation”, https://docs.soketi.app/.",
    "Uber Technologies, “Fx – A dependency injection system for Go”, https://uber-go.github.io/fx/.",
    "A. Cockburn, “Hexagonal Architecture”, https://alistair.cockburn.us/hexagonal-architecture/, 2005.",
    "R. C. Martin, Clean Architecture: A Craftsman’s Guide to Software Structure and Design, Prentice Hall, 2017.",
    "Meta Open Source, “React Documentation”, https://react.dev/.",
    "Mantine, “Mantine UI Documentation”, https://mantine.dev/.",
    "The PostgreSQL Global Development Group, “PostgreSQL 17 Documentation”, https://www.postgresql.org/docs/17/.",
    "Amazon Web Services, “Amazon S3 API Reference”, https://docs.aws.amazon.com/AmazonS3/latest/API/.",
    "D. Hardt, “The OAuth 2.0 Authorization Framework”, RFC 6749, IETF, 2012.",
    "N. Sakimura, J. Bradley & N. Agarwal, “Proof Key for Code Exchange by OAuth Public Clients”, RFC 7636, IETF, 2015.",
    "N. Sakimura et al., “OpenID Connect Core 1.0”, OpenID Foundation, https://openid.net/specs/openid-connect-core-1_0.html, 2014.",
    "M. Jones, J. Bradley & N. Sakimura, “JSON Web Token (JWT)”, RFC 7519, IETF, 2015.",
    "Google, “Gmail API – Push Notifications”, https://developers.google.com/gmail/api/guides/push.",
    "Google Cloud, “Pub/Sub Documentation”, https://cloud.google.com/pubsub/docs.",
    "A. Vaswani et al., “Attention Is All You Need”, Advances in Neural Information Processing Systems 30 (NeurIPS), 2017.",
    "DeepSeek, “DeepSeek API Docs”, https://api-docs.deepseek.com/.",
    "Docker Inc., “Docker Documentation”, https://docs.docker.com/.",
    "F5 / NGINX, “nginx documentation”, https://nginx.org/en/docs/.",
    "Cloudflare, “Cloudflare Tunnel”, https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/.",
    "E. F. Codd, “A Relational Model of Data for Large Shared Data Banks”, Communications of the ACM, vol. 13, no. 6, pp. 377–387, 1970.",
]
for i, r in enumerate(refs, start=1):
    p = para(f"[{i}] {r}", align=WD_ALIGN_PARAGRAPH.LEFT)
    p.paragraph_format.left_indent = Cm(0.8)
    p.paragraph_format.first_line_indent = Cm(-0.8)

# ================= PHỤ LỤC
front_title("PHỤ LỤC")
render_chapter("phu-luc.md")

out = sys.argv[1] if len(sys.argv) > 1 else "khung-bao-cao-DATN.docx"
doc.save(out)
print("saved", out)
