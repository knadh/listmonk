#!/usr/bin/env python3

import argparse
import shutil
import xml.etree.ElementTree as ET
from pathlib import Path
from urllib.request import urlretrieve
from zipfile import ZipFile

FEATHER_ZIP_URL = "https://github.com/feathericons/feather/archive/refs/heads/master.zip"


def setup_files(vendor_dir: Path) -> Path:
    feather_dir = vendor_dir / "feather"
    icons_dir = feather_dir / "icons"
    if icons_dir.exists():
        return feather_dir

    vendor_dir.mkdir(parents=True, exist_ok=True)

    zip_path = vendor_dir / "feather.zip"
    extract_dir = vendor_dir / "_feather_extract"

    if extract_dir.exists():
        shutil.rmtree(extract_dir)

    print("Downloading Feather icons...")
    urlretrieve(FEATHER_ZIP_URL, zip_path)

    print("Extracting Feather icons...")
    with ZipFile(zip_path) as z:
        z.extractall(extract_dir)

    extracted_root = next(extract_dir.iterdir())

    if feather_dir.exists():
        shutil.rmtree(feather_dir)

    shutil.move(str(extracted_root), str(feather_dir))

    zip_path.unlink()
    shutil.rmtree(extract_dir)

    return feather_dir


def read_icons(path: Path) -> list[str]:
    return [
        line.strip()
        for line in path.read_text(encoding="utf-8").splitlines()
        if line.strip()
    ]


def svg_to_symbol(feather_dir: Path, icon_name: str) -> str:
    path = feather_dir / "icons" / f"{icon_name}.svg"

    if not path.exists():
        raise FileNotFoundError(f"Missing Feather icon: {icon_name}")

    root = ET.parse(path).getroot()
    for elem in root.iter():
        elem.tag = elem.tag.rsplit("}", 1)[-1]

    viewbox = root.attrib.get("viewBox", "0 0 24 24")

    attrs = {
        "id": f"icon-{icon_name}",
        "viewBox": viewbox,
        "fill": root.attrib.get("fill", "none"),
        "stroke": root.attrib.get("stroke", "currentColor"),
        "stroke-width": root.attrib.get("stroke-width", "2"),
        "stroke-linecap": root.attrib.get("stroke-linecap", "round"),
        "stroke-linejoin": root.attrib.get("stroke-linejoin", "round"),
    }

    attr_str = " ".join(f'{k}="{v}"' for k, v in attrs.items())
    children = "".join(ET.tostring(child, encoding="unicode")
                       for child in root)

    return f"  <symbol {attr_str}>\n    {children}\n  </symbol>"


def build_sprite(vendor_dir: Path, icons_path: Path, out_path: Path) -> None:
    feather_dir = setup_files(vendor_dir)
    icons = read_icons(icons_path)

    out_path.parent.mkdir(parents=True, exist_ok=True)
    symbols = "\n\n".join(svg_to_symbol(feather_dir, name) for name in icons)

    sprite = f'''<svg xmlns="http://www.w3.org/2000/svg" style="display:none">
{symbols}
</svg>
'''

    out_path.write_text(sprite, encoding="utf-8")

    print(f"Wrote {out_path}")


def parse_args() -> argparse.Namespace:
    base_dir = Path(__file__).resolve().parent
    parser = argparse.ArgumentParser(
        description="Build a Feather icons SVG sprite.")
    parser.add_argument(
        "--dir", type=Path, default=base_dir / ".icons", help="vendor directory"
    )
    parser.add_argument(
        "--icons", type=Path, default=base_dir / "icons.txt", help="icon list"
    )
    parser.add_argument(
        "--out", type=Path, default=base_dir / "icons.svg", help="sprite output file"
    )
    return parser.parse_args()


if __name__ == "__main__":
    args = parse_args()
    build_sprite(args.dir, args.icons, args.out)
