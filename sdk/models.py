from dataclasses import dataclass
from typing import Literal

LanguageType = Literal["c", "cpp", "java", "python", "go", "js"]

@dataclass
class SubmissionRequest:
    submission_id: str 
    bucket: str 
    language: str 
    source: str 
    testset_id: str


@dataclass
class JudgeEvent:
    type: str
    status: str
    stdout: str
    stderr: str
    details: str

    @classmethod
    def from_dict(cls, data: dict) -> "JudgeEvent":
        return cls(
            type=data.get("Type", ""),
            status=data.get("Status", ""),
            stdout=data.get("Stdout", ""),
            stderr=data.get("Stderr", ""),
            details=data.get("Details", "")
        )