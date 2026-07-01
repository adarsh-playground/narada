import re
from pypdf import PdfReader, PdfWriter

def split_pdf_by_text_markers(pdf_path, markers):
    reader = PdfReader(pdf_path)
    num_pages = len(reader.pages)

    # Step 1: Find the starting page for each marker
    marker_pages = {}

    for page_num in range(num_pages):
        # Extract text and normalize spaces to catch things like "Chapter   1"
        page_text = reader.pages[page_num].extract_text()
        if not page_text:
            continue
        page_text_clean = " ".join(page_text.split())

        for marker in markers:
            # Using regex for case-insensitive matching and flexible spacing
            pattern = re.compile(re.escape(marker), re.IGNORECASE)
            if pattern.search(page_text_clean):
                # Only record the first time a marker appears
                if marker not in marker_pages:
                    marker_pages[marker] = page_num
                    print(f"Found '{marker}' on Page {page_num + 1}")

    # Step 2: Slice and save the PDFs based on the found pages
    for i in range(len(markers)):
        current_marker = markers[i]

        if current_marker not in marker_pages:
            print(f"Warning: Could not find '{current_marker}' in the document. Skipping.")
            continue

        start_page = marker_pages[current_marker]

        # If there's a next marker, stop there; otherwise, go to the end of the file
        if i + 1 < len(markers) and markers[i + 1] in marker_pages:
            end_page = marker_pages[markers[i + 1]]
        else:
            end_page = num_pages

        # Avoid creating empty files if markers happen to be on the same page
        if start_page >= end_page and i + 1 < len(markers):
            end_page = start_page + 1

        # Write the selected pages to a new PDF
        writer = PdfWriter()
        for page in range(start_page, end_page):
            writer.add_page(reader.pages[page])

        clean_filename = current_marker.lower().replace(" ", "_") + ".pdf"
        with open(clean_filename, "wb") as f:
            writer.write(f)

        print(f"Saved: {clean_filename} (Pages {start_page + 1} to {end_page})")

# Define your chapter list sequentially
chapter_markers = [
    "Chapter 1",
    "Chapter 2",
    "Chapter 3",
    "Chapter 4",
    "Chapter 5",
    "Chapter 6",
    "Chapter 7",
    "Chapter 8",
    "Chapter 9",
    "Chapter 10",
    "Chapter 11",
    "Chapter 12",
    "Chapter 13",
    "Chapter 14",
    "Chapter 15",
    "Chapter 16",
    "Chapter 17",
    "Chapter 18",
    "Chapter 19"
]

# Run the split
split_pdf_by_text_markers("Holy Geeta by Swami Chinmayana.pdf", chapter_markers)