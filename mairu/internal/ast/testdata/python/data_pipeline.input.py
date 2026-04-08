import json
import logging
from pathlib import Path
from typing import List, Dict

logger = logging.getLogger(__name__)


def load_records(path: str) -> List[Dict]:
    with open(path, "r") as f:
        return json.load(f)


def validate(records: List[Dict]) -> List[Dict]:
    valid = []
    for record in records:
        if "id" not in record or "value" not in record:
            logger.warning("Skipping invalid record: %s", record)
            continue
        valid.append(record)
    return valid


def transform(records: List[Dict]) -> List[Dict]:
    return [
        {"id": r["id"], "score": compute_score(r["value"])}
        for r in records
    ]


def compute_score(value: float) -> float:
    return round(value * 0.85 + 10, 2)


def save_results(records: List[Dict], output_path: str) -> None:
    Path(output_path).parent.mkdir(parents=True, exist_ok=True)
    with open(output_path, "w") as f:
        json.dump(records, f, indent=2)


def run_pipeline(input_path: str, output_path: str) -> int:
    raw = load_records(input_path)
    valid = validate(raw)
    transformed = transform(valid)
    save_results(transformed, output_path)
    return len(transformed)
