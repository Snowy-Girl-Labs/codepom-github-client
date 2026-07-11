from pydantic import BaseModel, Field
from typing import Optional

class SonarQubeIssuePayload(BaseModel):
    issue_key: str = Field(..., alias="issueKey")
    project_key: str = Field(..., alias="projectKey")
    file_path: str = Field(..., alias="filePath")
    line_number: int = Field(..., alias="lineNumber")
    rule_key: str = Field(..., alias="ruleKey")
    message: str
    severity: str

    class Config:
        populate_by_name = True

class TriageResult(BaseModel):
    is_valid: bool = Field(..., description="Whether the issue is valid and needs fixing")
    assignee_email: Optional[str] = Field(None, description="SCM blame author email")
    github_issue_created: bool = False
    github_issue_number: Optional[int] = None
    reasoning: str
