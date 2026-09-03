#!/usr/bin/env python3
"""Extract Swami Chinmayananda's verse translations and commentary from the PDF.

The book uses BookAntiqua-Italic for translations and regular Book Antiqua for
commentary. Consecutive italic verse blocks followed by one prose block (for
example 1.4-1.6) are represented as one commentary passage linked to every
verse in that group.
"""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path

import pdfplumber

# Some pages print "36 .", "41" (without a period), or prefix the number with
# "The Blessed Lord said:". A verse number must occur near the start so years
# and numbered references inside the translation cannot be mistaken for it.
VERSE_START = re.compile(
    r"^(?:(?:Dhritarashtra|Sanjaya|Arjuna|The Blessed Lord)\s+(?:said|says)\s*:\s*)?"
    r"(\d+)(?:\s*[-–]\s*(\d+))?\s*\.?\s+",
    re.IGNORECASE,
)
CHAPTER_MARKER = re.compile(r"^Chapter\s+(\d+)\s*$", re.IGNORECASE)


def normalize(lines: list[str]) -> str:
    text = " ".join(line.strip() for line in lines if line.strip())
    text = re.sub(r"\s+", " ", text)
    text = re.sub(r"(?<=\w)-\s+(?=\w)", "", text)
    return text.strip()


def is_italic(line: dict) -> bool:
    visible = [char for char in line["chars"] if char.get("text", "").strip()]
    # Sanskrit terms inside translations are frequently set in regular or
    # bold small caps, so requiring an entirely italic line drops verses.
    # Commentary may italicize an isolated term, but not this much of a line.
    return bool(visible) and sum("Italic" in char.get("fontname", "") for char in visible) / len(visible) > 0.35


def all_chapter_lines(pdf_path: Path) -> dict[int, list[tuple[str, bool]]]:
    collected: dict[int, list[tuple[str, bool]]] = {chapter: [] for chapter in range(1, 19)}
    active: int | None = None
    with pdfplumber.open(pdf_path) as pdf:
        for page in pdf.pages:
            lines = page.extract_text_lines(return_chars=True)
            for line in lines:
                text = line["text"].strip()
                marker = CHAPTER_MARKER.fullmatch(text)
                if marker:
                    number = int(marker.group(1))
                    active = number if 1 <= number <= 18 else None
                    continue
                if active is None:
                    continue
                # Running title and bare printed page numbers are not content.
                if text == "Holy Geeta by Swami Chinmayananda" or text.isdigit():
                    continue
                collected[active].append((text, is_italic(line)))
    return collected


def chapter_lines(pdf_path: Path, chapter: int) -> list[tuple[str, bool]]:
    return all_chapter_lines(pdf_path)[chapter]


def parse(lines: list[tuple[str, bool]], chapter: int) -> dict:
    translations: list[dict] = []
    passages: list[dict] = []
    pending_verses: list[int] = []
    commentary_lines: list[str] = []
    italic_lines: list[str] = []

    def flush_translation() -> None:
        nonlocal italic_lines
        if not italic_lines:
            return
        text = normalize(italic_lines)
        match = VERSE_START.search(text)
        if not match:
            # Chapter introductions and colophons are italic too, but are not
            # verse translations.
            italic_lines = []
            return
        first = int(match.group(1))
        last = int(match.group(2) or first)
        if last < first:
            raise ValueError(f"invalid verse range {first}-{last}")
        for verse in range(first, last + 1):
            translations.append({"verse_number": str(verse), "text": text})
            pending_verses.append(verse)
        italic_lines = []

    def flush_commentary() -> None:
        nonlocal commentary_lines, pending_verses
        text = normalize(commentary_lines)
        if text and pending_verses:
            passages.append({
                "sequence_number": len(passages) + 1,
                "verse_numbers": [str(v) for v in pending_verses],
                "citation_label": f"BG {chapter}." + (str(pending_verses[0]) if len(pending_verses) == 1 else f"{pending_verses[0]}-{pending_verses[-1]}"),
                "text": text,
            })
        commentary_lines = []
        pending_verses = []

    started = False
    forced_upper_translation = False
    for text, italic in lines:
        starts_verse = VERSE_START.search(text) is not None
        letters = "".join(char for char in text if char.isalpha())
        all_upper = bool(letters) and letters == letters.upper()
        if starts_verse and not italic:
            # A few translations (notably BG 11.22) are printed in all caps
            # rather than italics.
            forced_upper_translation = all_upper
            italic = True
        elif forced_upper_translation and all_upper:
            italic = True
        else:
            forced_upper_translation = False
        if italic:
            if commentary_lines:
                flush_commentary()
            # A new numbered italic line starts a new translation block.
            if italic_lines and starts_verse:
                flush_translation()
            italic_lines.append(text)
            started = True
        elif started:
            flush_translation()
            commentary_lines.append(text)
    flush_translation()
    flush_commentary()

    # The edition occasionally uses overlapping printed ranges for a verse
    # split across two blocks (1.20-21 followed by 1.21-22). The later block
    # owns the shared canonical verse; retain the printed label in its text.
    translations = list({item["verse_number"]: item for item in translations}.values())
    claimed_later: set[str] = set()
    for passage in reversed(passages):
        passage["verse_numbers"] = [v for v in passage["verse_numbers"] if v not in claimed_later]
        claimed_later.update(passage["verse_numbers"])
    passages = [passage for passage in passages if passage["verse_numbers"]]
    for sequence, passage in enumerate(passages, 1):
        passage["sequence_number"] = sequence

    return {
        "chapter": chapter,
        "source": {
            "name": "Swami Chinmayananda",
            "publication": "The Holy Geeta",
            "language": "English",
        },
        "translations": translations,
        "commentary_passages": passages,
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--pdf", type=Path, default=Path(__file__).with_name("Holy Geeta by Swami Chinmayana.pdf"))
    parser.add_argument("--chapter", default="all", help="chapter number or 'all' (default)")
    parser.add_argument("--output", type=Path, help="output file for a single chapter")
    parser.add_argument("--output-dir", type=Path, default=Path(__file__).parent)
    args = parser.parse_args()
    chapters = list(range(1, 19)) if args.chapter.lower() == "all" else [int(args.chapter)]
    if any(chapter < 1 or chapter > 18 for chapter in chapters):
        parser.error("chapter must be 1-18 or 'all'")
    if args.output and len(chapters) != 1:
        parser.error("--output can only be used with one chapter")

    extracted = all_chapter_lines(args.pdf)
    canonical_verses = json.loads((Path(__file__).parents[2] / "verse.json").read_text(encoding="utf-8"))
    args.output_dir.mkdir(parents=True, exist_ok=True)
    for chapter in chapters:
        result = parse(extracted[chapter], chapter)
        expected = {str(verse["verse_number"]) for verse in canonical_verses if verse["chapter_number"] == chapter}
        actual = {item["verse_number"] for item in result["translations"]}
        if actual != expected:
            key = lambda value: int(value)
            raise ValueError(
                f"chapter {chapter} verse mismatch: "
                f"missing={sorted(expected - actual, key=key)}, extra={sorted(actual - expected, key=key)}"
            )
        output = args.output or args.output_dir / f"chapter_{chapter}.json"
        output.write_text(json.dumps(result, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        print(f"wrote {len(result['translations'])} translations and {len(result['commentary_passages'])} commentary passages to {output}")


if __name__ == "__main__":
    main()
