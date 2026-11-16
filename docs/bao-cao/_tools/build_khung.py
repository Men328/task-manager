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
    rfonts = run_or_style.element.get_or_add_rPr().get_or_add_rFonts()
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


def page_number_footer(section, fmt):
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
    pg.set(qn("w:start"), "1")


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
    doc.add_paragraph().paragraph_format.space_after = Pt(0)
    return t



# ---------------------------------------------------------------- nội dung chương từ file .md
NOI_DUNG_DIR = Path(__file__).resolve().parent.parent / "noi-dung"
HINH_DIR = Path(__file__).resolve().parent.parent / "hinh"


def add_rich_text(p, text, size=None):
    """Hỗ trợ **đậm** và *nghiêng* đơn giản trong một đoạn."""
    import re as _re
    for part in _re.split(r"(\*\*[^*]+\*\*|\*[^*]+\*)", text):
        if not part:
            continue
        if part.startswith("**"):
            r = p.add_run(part[2:-2])
            set_font(r, size, bold=True)
        elif part.startswith("*"):
            r = p.add_run(part[1:-1])
            set_font(r, size, italic=True)
        else:
            r = p.add_run(part)
            set_font(r, size)


def content_table(caption, rows):
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

# Bỏ header/footer của mẫu (footer chứa mã biểu mẫu "BM.ĐT.19.06 ...") khỏi trang bìa.
body_sect = doc.element.body.find(qn("w:sectPr"))
for tag in ("w:headerReference", "w:footerReference"):
    for ref in body_sect.findall(qn(tag)):
        body_sect.remove(ref)

# ---------------------------------------------------------------- Phần đầu (đánh số La Mã)
front = doc.add_section(WD_SECTION.NEW_PAGE)
setup_page(front)
page_number_footer(front, "lowerRoman")
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
box(
    can=["Bảng từ viết tắt sắp xếp A→Z; điền đầy đủ tên tiếng Anh + nghĩa tiếng Việt."],
    nguon=[f"{XLSX} sheet '4. Thuật ngữ'", f"{D03} §2 Thuật ngữ"],
)
abbr = ["API", "CRUD", "CSDL", "DI", "ERD", "gRPC", "HTTP", "JSON", "JWT", "LLM", "OAuth",
        "OIDC", "PKCE", "REST", "RRULE", "S3", "SPA", "SQL", "UC", "UI", "UUID", "WS"]
t = table("", ["Từ viết tắt", "Tiếng Anh", "Nghĩa tiếng Việt"], rows=len(abbr))
for i, a in enumerate(abbr, start=1):
    t.rows[i].cells[0].text = a
    set_font(t.rows[i].cells[0].paragraphs[0].runs[0], 12)

# ---------------------------------------------------------------- Phần chính (số Ả Rập)
main = doc.add_section(WD_SECTION.NEW_PAGE)
setup_page(main)
page_number_footer(main, "decimal")

# ================= CHƯƠNG 1, 2 (đã viết nội dung: docs/bao-cao/noi-dung/)
render_chapter("chuong-1.md", first=True)
render_chapter("chuong-2.md")

# ================= CHƯƠNG 3 (đã viết nội dung)
render_chapter("chuong-3.md")

# ================= CHƯƠNG 4
h1("CHƯƠNG 4: THIẾT KẾ HỆ THỐNG")

h2("4.1. Thiết kế kiến trúc tổng thể")
h3("4.1.1. Kiến trúc triển khai")
box(can=["Sơ đồ container: nginx là cổng vào duy nhất, mỗi service 2 container (HTTP gateway + gRPC), hạ tầng."],
    nguon=[f"{D02} §1", "deployments/docker/docker-compose.yml"])
figure("Hình 4.1: Sơ đồ kiến trúc triển khai hệ thống")
h3("4.1.2. Kiến trúc phân lớp logic")
box(can=["Client – Edge – Gateway – Domain – Async – Data."], nguon=[f"{D02} §2"])
figure("Hình 4.2: Sơ đồ kiến trúc phân lớp logic")
h3("4.1.3. Danh sách dịch vụ và trách nhiệm")
box(can=["Bảng 13 service: port gRPC/HTTP, chức năng, kho dữ liệu."], nguon=[f"{D01} §4.2"])
table("Bảng 4.1: Danh sách các dịch vụ trong hệ thống", ["STT", "Dịch vụ", "Cổng gRPC / HTTP", "Chức năng", "Lưu trữ"], rows=3)
h3("4.1.4. Cấu trúc bên trong một dịch vụ")
box(can=["cmd/grpc, cmd/http, internal/{handler, service, model, repository, dependency, config}; quy tắc quality.json."],
    nguon=[f"{D01} §6", "quality.json"])
h3("4.1.5. Các quyết định kiến trúc")
box(can=["Bảng quyết định – lý do – hệ quả."], nguon=[f"{D02} §5"])
table("Bảng 4.2: Các quyết định kiến trúc chính", ["STT", "Quyết định", "Lý do / hệ quả"], rows=3)

h2("4.2. Thiết kế giao tiếp giữa các dịch vụ")
h3("4.2.1. Ma trận gọi dịch vụ đồng bộ (gRPC)")
box(can=["Bảng caller – callee – RPC – thời điểm – xử lý lỗi; sơ đồ gọi gRPC."],
    nguon=[f"{D05} §1, §1.1, §1.2"])
table("Bảng 4.3: Ma trận gọi gRPC giữa các dịch vụ", ["STT", "Bên gọi", "Bên được gọi", "RPC", "Khi nào", "Xử lý lỗi"], rows=3)
figure("Hình 4.3: Sơ đồ phụ thuộc gRPC giữa các dịch vụ")
h3("4.2.2. Giao tiếp bất đồng bộ và thời gian thực")
box(can=["Kafka topic chat.messages (key, partition, consumer group); kênh Soketi và event."],
    nguon=[f"{D05} §4.5, §4.6"])
h3("4.2.3. Tích hợp hệ thống bên ngoài")
box(can=["Bảng gọi Google OAuth, Gmail, Pub/Sub, DeepSeek, Soketi (method + URL + mục đích)."],
    nguon=[f"{D05} §2"])
table("Bảng 4.4: Danh sách tích hợp hệ thống bên ngoài", ["STT", "Bên gọi", "Đích", "API", "Mục đích"], rows=3)

h2("4.3. Thiết kế API")
h3("4.3.1. Quy ước thiết kế API")
box(can=["Prefix nginx /api/<svc>, path /v1/<resource>, JSON lowerCamelCase, query theo tên field proto."],
    nguon=[f"{D05} §3", README + " mục Quy ước"])
h3("4.3.2. Danh sách API theo dịch vụ")
box(can=["Mỗi dịch vụ 1 bảng: Method – Path – RPC – Xác thực. (Có thể đưa bảng đầy đủ vào Phụ lục.)"],
    nguon=[f"{D05} §3.1 → §3.13"])
table("Bảng 4.5: Danh sách API của hệ thống", ["Dịch vụ", "Method", "Path", "RPC", "Xác thực"], rows=3)
h3("4.3.3. Bộ mã lỗi chuẩn")
box(can=["error_codes.json, ErrorInfo.reason, FE dịch vi/en; ví dụ response lỗi."],
    nguon=[README + " mục Bộ mã lỗi chuẩn", "common/errorcode/error_codes.json"])
h3("4.3.4. Thiết kế xác thực và phân quyền")
box(can=["Session JWT, gateway verify (identity/notification/social/message), uỷ quyền kênh Soketi; "
         "nêu thẳng giới hạn hiện tại (một số gateway nhận profile_id từ client)."],
    nguon=[f"{D02} §3"])

h2("4.4. Thiết kế cơ sở dữ liệu")
h3("4.4.1. Sơ đồ thực thể liên kết")
box(can=["ERD toàn hệ thống; giải thích FK chéo schema (profile CASCADE, project RESTRICT), liên kết đa hình."],
    nguon=[f"{D01} §5", "design/db_schema.dbml", "deployments/migrations/"])
figure("Hình 4.4: Sơ đồ thực thể liên kết (ERD)")
h3("4.4.2. Danh sách schema và bảng")
box(can=["Bảng: schema – bảng – service sở hữu – mô tả."], nguon=[f"{D05} §4.1, §4.3"])
table("Bảng 4.6: Danh sách các bảng trong hệ thống", ["STT", "Schema", "Bảng", "Service sở hữu", "Mô tả"], rows=3)

tables_db = [
    ("identity.profiles", "000001_init_identity"),
    ("identity.auth_providers", "000001_init_identity"),
    ("project.projects", "000011_init_project"),
    ("task.task_statuses", "000002_init_task"),
    ("task.status_transitions", "000002_init_task"),
    ("task.tasks", "000002, 000011–000014"),
    ("task.task_status_logs", "000002_init_task"),
    ("calendar.schedules", "000006_init_calendar"),
    ("event.events", "000007_init_event"),
    ("backlog.backlogs", "000008_init_backlog"),
    ("attachment.attachments", "000010_init_attachment"),
    ("notification.notices", "000009, 000016"),
    ("mail_provider.sessions", "000005_init_mail_provider"),
    ("mail_provider.noti_indexes", "000005_init_mail_provider"),
    ("social.friendships", "000015_init_social"),
    ("message.conversations", "000017_init_message"),
    ("message.messages", "000017_init_message"),
]
for i, (name, mig) in enumerate(tables_db, start=3):
    h3(f"4.4.{i}. Bảng {name}")
    box(can=["Mục đích bảng; bảng cấu trúc cột; index & ràng buộc quan trọng."],
        nguon=[f"deployments/migrations/{mig}.up.sql", f"{D05} §4.3"])
    table(f"Bảng 4.{4 + i}: Cấu trúc bảng {name}", TABLE_COLS, rows=2)

h3(f"4.4.{3 + len(tables_db)}. Thiết kế lưu trữ đối tượng và hàng đợi")
box(can=["Quy ước object key S3 (email gốc, attachment); cấu trúc message Kafka (ChatMessage)."],
    nguon=[f"{D05} §4.4, §4.5", "common/proto/message/v1/message.proto"])

h2("4.5. Thiết kế xử lý nghiệp vụ trọng tâm")
for code, name, src in [
    ("4.5.1", "Vòng đời task theo luật chuyển trạng thái (lifecycle allowlist)", f"{D03} §3.3"),
    ("4.5.2", "Phân loại email và điều phối (dispatch)", f"{D03} §3.5; service/mail-provider/README.md"),
    ("4.5.3", "Nhắc việc và sinh task lặp lại", f"{D03} §3.3"),
    ("4.5.4", "Nhắn tin: realtime trước, lưu bất đồng bộ, idempotent", f"{D03} §3.10; {D05} §4.5"),
]:
    h3(f"{code}. {name}")
    box(can=["Thuật toán/luật, sơ đồ hoạt động (activity) hoặc bảng quyết định, các trường hợp biên."], nguon=[src])
table("Bảng 4.24: Bảng quyết định điều phối email sau phân loại", ["Category", "Điều kiện", "Đích", "Lý do khi vào backlog"], rows=4)

h2("4.6. Thiết kế giao diện ứng dụng")
box(can=["Mỗi trang: ảnh chụp màn hình + mô tả thành phần + thao tác chính + service được gọi."],
    nguon=[f"{D01} §4.1 (bảng trang – chức năng – service)", "frontend/src/pages/", "design/ui.png"],
    luu_y=["Chụp ảnh với dữ liệu demo (make seed-demo); thống nhất ngôn ngữ giao diện (tiếng Việt)."])
pages = [
    "Trang đăng nhập", "Trang bảng công việc (Kanban)", "Các chế độ xem Bảng / Danh sách / Dòng thời gian",
    "Cửa sổ chi tiết task", "Trang việc hôm nay", "Trang quản lý dự án", "Trang lịch",
    "Trang sự kiện", "Trang backlog", "Trang báo cáo", "Thông báo (chuông)", "Trang bạn bè", "Trang tin nhắn",
]
for i, name in enumerate(pages, start=1):
    h3(f"4.6.{i}. {name}")
    figure(f"Hình 4.{4 + i}: {name}")

# ================= CHƯƠNG 5
h1("CHƯƠNG 5: TRIỂN KHAI VÀ KIỂM THỬ HỆ THỐNG")
box(luu_y=["Báo cáo mẫu chỉ có 'Kiểm thử'; đồ án này bổ sung phần Triển khai vì hệ thống nhiều thành phần."])

h2("5.1. Triển khai hệ thống")
h3("5.1.1. Môi trường phát triển")
box(can=["Bảng: hệ điều hành, Go, Node, Docker, IDE, buf, công cụ kiểm thử."],
    nguon=[README + " mục Yêu cầu", "docs/getting-started.md"])
table("Bảng 5.1: Môi trường phát triển", ["STT", "Thành phần", "Phiên bản", "Ghi chú"], rows=3)
h3("5.1.2. Tổ chức mã nguồn")
box(can=["Monorepo: common/, service/, frontend/, deployments/, scripts/, tools/; go.work."],
    nguon=[README + " mục Cấu trúc"])
h3("5.1.3. Triển khai bằng Docker Compose")
box(can=["Các lệnh make up/up-core, migrate, scale worker; danh sách container."],
    nguon=["deployments/docker/README.md", "Makefile", f"{D05} §6"])
h3("5.1.4. Đưa hệ thống ra Internet")
box(can=["Cloudflare Tunnel, webhook Pub/Sub cần HTTPS công khai."],
    nguon=["deployments/docker/README.md"])
h3("5.1.5. Kiểm soát chất lượng mã nguồn")
box(can=["make quality-all: gofmt, go vet, luật kiến trúc (quality.json), kiểm tra mã lỗi; unit test Go & Vitest."],
    nguon=[README + " mục Kiểm tra chất lượng", "quality.json"])

h2("5.2. Kiểm thử chức năng")
box(can=["Mỗi nhóm 1 bảng test case theo đúng mẫu: Chức năng – Dữ liệu đầu vào – Kết quả mong đợi – "
         "Kết quả thực tế – Trạng thái. Lấy test case từ tiêu chí chấp nhận (AC) và mã lỗi."],
    nguon=[f"{D04} (AC của từng US)", "common/errorcode/error_codes.json", "các file *_test.go"])
groups = [
    ("Kiểm thử chức năng xác thực người dùng", "US-01..04"),
    ("Kiểm thử chức năng quản lý task và vòng đời trạng thái", "US-07..17"),
    ("Kiểm thử chức năng quản lý dự án", "US-05..06"),
    ("Kiểm thử chức năng lịch và sự kiện", "US-18..19"),
    ("Kiểm thử chức năng tự động xử lý email và backlog", "US-20..21"),
    ("Kiểm thử chức năng tệp đính kèm", "US-22"),
    ("Kiểm thử chức năng báo cáo", "US-23..24"),
    ("Kiểm thử chức năng thông báo realtime", "US-25"),
    ("Kiểm thử chức năng bạn bè và nhắn tin", "US-26..27"),
]
for i, (name, us) in enumerate(groups, start=1):
    h3(f"5.2.{i}. {name}")
    box(nguon=[f"{D04} {us}"])
    table(f"Bảng 5.{i + 1}: Kết quả {name[0].lower() + name[1:]}", TEST_COLS, rows=3)

h2("5.3. Kiểm thử phi chức năng")
box(can=["Độ tin cậy: tắt Soketi / notification vẫn tạo được task; tắt DeepSeek → luật dự phòng; "
         "worker chết rồi chạy lại không mất/trùng tin; scale worker theo partition.",
         "Bảo mật: xin kênh Soketi người khác bị từ chối; token sai bị 401."],
    nguon=[f"{D03} §4", f"{D02} §5"])
table("Bảng 5.11: Kết quả kiểm thử phi chức năng", ["STT", "Kịch bản", "Cách thực hiện", "Kết quả mong đợi", "Kết quả thực tế", "Trạng thái"], rows=3)

h2("5.4. Đánh giá kết quả")
box(can=["Tổng hợp số test case pass/fail, mức độ hoàn thành yêu cầu ở Bảng 3.1 / 3.2."])

# ================= KẾT LUẬN
front_title("KẾT LUẬN")
para("1. Kết quả đạt được", bold=True)
box(can=["Đối chiếu từng mục tiêu ở 1.2 → kết quả; kiến thức/kỹ năng thu được."],
    nguon=[f"{D04} epic E1–E9"])
para("2. Hạn chế của đồ án", bold=True)
box(can=["Một số gateway chưa verify JWT; chưa dedupe email theo messageId; chỉ Gmail; "
         "chưa có app di động; chưa có giám sát/log tập trung; lịch/sự kiện chưa lặp, chưa đồng bộ Google Calendar."],
    nguon=[f"{D02} §3", "service/mail-provider/README.md §Giới hạn đã biết", f"{XLSX} sheet '5. Hỏi đáp'"])
para("3. Hướng phát triển", bold=True)
box(can=["Xác thực tập trung ở gateway; observability (metrics, tracing); hỗ trợ Outlook; mobile app; "
         "triển khai Kubernetes/cloud; cộng tác nhóm; đồng bộ Google Calendar."])

# ================= TÀI LIỆU THAM KHẢO
front_title("DANH MỤC TÀI LIỆU THAM KHẢO")
box(can=["Trình bày theo chuẩn IEEE như mẫu: [n] Tác giả, “Tên tài liệu”, nguồn/URL, năm."],
    luu_y=["Số thứ tự [n] khớp trích dẫn trong Chương 2; chương sau trích dẫn thêm thì bổ sung vào cuối danh sách. Kiểm tra lại URL trước khi nộp."])
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
para("Phụ lục A. Hướng dẫn cài đặt và chạy hệ thống", bold=True)
box(nguon=["docs/getting-started.md", README + " mục Quickstart"])
para("Phụ lục B. Danh sách đầy đủ API", bold=True)
box(nguon=[f"{D05} §3"])
para("Phụ lục C. Danh sách biến môi trường", bold=True)
box(nguon=["README.md của từng service", f"{D05} §6"])
para("Phụ lục D. Bộ mã lỗi chuẩn", bold=True)
box(nguon=["common/errorcode/error_codes.json"])
box(luu_y=["Phụ lục là tuỳ chọn (báo cáo mẫu không có); giúp chương 4 gọn hơn."])

out = sys.argv[1] if len(sys.argv) > 1 else "khung-bao-cao-DATN.docx"
doc.save(out)
print("saved", out)
