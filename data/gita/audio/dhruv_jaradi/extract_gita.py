
from datasets import load_dataset, Audio
from pathlib import Path
import soundfile as sf
import io

ds = load_dataset(
    "JDhruv14/Bhagavad-Gita_Audio",
    split="train"
)

# Avoid TorchCodec entirely
ds = ds.cast_column("audio", Audio(decode=False))

output_dir = Path("gita_audio")
output_dir.mkdir(parents=True, exist_ok=True)

count = 0

for row in ds:
    verse_id = row["shloka_id"]
    audio = row["audio"]

    if audio["bytes"] is None:
        raise ValueError(f"Missing audio bytes: {verse_id}")

    audio_data, sample_rate = sf.read(
        io.BytesIO(audio["bytes"])
    )

    output_file = output_dir / f"{verse_id}.wav"

    sf.write(output_file, audio_data, sample_rate)

    count += 1
    print(f"[{count}/701] {output_file}")

print(f"Successfully extracted {count} recordings")
