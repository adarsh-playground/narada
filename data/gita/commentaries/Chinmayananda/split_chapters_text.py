import re
from pypdf import PdfReader

def split_pdf_by_exact_word(pdf_path, markers):
    print(f"Reading and extracting text from {pdf_path}...")
    reader = PdfReader(pdf_path)

    text_list = []
    for page in reader.pages:
        page_text = page.extract_text()
        if page_text:
            text_list.append(page_text)

    full_text = "\n".join(text_list)
    print("Text extraction complete. Splitting by mid-page chapter markers...")

    # FIX: Sort markers by length descending so longer strings ("Chapter 10")
    # are evaluated before shorter strings ("Chapter 1") to prevent greedy partial matches.
    sorted_markers = sorted(markers, key=len, reverse=True)

    # FIX: Wrap each marker in \b (word boundary) patterns to prevent "Chapter 1" matching "Chapter 10"
    pattern_parts = [rf"\b{re.escape(m)}\b" for m in sorted_markers]
    pattern = "(" + "|".join(pattern_parts) + ")"

    # We use re.split, but since we modified the pattern to include \b,
    # we need a clean way to identify if a chunk matches our original markers list.
    chunks = re.split(pattern, full_text, flags=re.IGNORECASE)

    current_marker = "preamble"
    chapter_content = ""

    # Normalize markers list for easy verification in the loop
    normalized_markers = {m.lower().strip() for m in markers}

    for chunk in chunks:
        chunk_clean = chunk.lower().strip() if chunk else ""
        if chunk_clean in normalized_markers:
            # Save the accumulated text of the previous chapter
            if chapter_content.strip() and current_marker != "preamble":
                save_chapter(current_marker, chapter_content)

            # Keep the original casing of the marker for the filename
            current_marker = chunk.strip()
            chapter_content = ""
        else:
            if chunk:
                chapter_content += chunk

    # Save the very last chapter
    if chapter_content.strip() and current_marker != "preamble":
        save_chapter(current_marker, chapter_content)

def save_chapter(marker_name, content):
    safe_name = "".join(c for c in marker_name if c.isalnum() or c in (' ', '_', '-')).rstrip()
    filename = safe_name.lower().replace(" ", "_") + ".txt"

    with open(filename, "w", encoding="utf-8") as f:
        f.write(marker_name + "\n" + content.lstrip())
    print(f"Saved: {filename}")

# Expanded list to cover all 18 chapters of the Gita
chapter_markers = [f"Chapter {i}" for i in range(1, 19)]

# Run the updated split
split_pdf_by_exact_word("Holy Geeta by Swami Chinmayana.pdf", chapter_markers)
