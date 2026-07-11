#!/usr/bin/env python3
import sys
import json
from agent.schemas import SonarQubeIssuePayload, TriageResult

def main():
    # Read payload from stdin or command-line arg
    if len(sys.argv) > 1:
        raw_input = sys.argv[1]
    else:
        raw_input = sys.stdin.read()

    try:
        data = json.loads(raw_input)
        issue = SonarQubeIssuePayload(**data)
    except Exception as e:
        print(json.dumps({"error": f"Failed to parse input schema: {str(e)}"}), file=sys.stderr)
        sys.exit(1)

    # Output mock triage results for demonstration
    result = TriageResult(
        is_valid=True,
        assignee_email="dev@example.com",
        github_issue_created=True,
        github_issue_number=42,
        reasoning=f"Issue {issue.issue_key} is valid on rule {issue.rule_key} at {issue.file_path}:{issue.line_number}."
    )

    # Return structured output on stdout
    print(result.model_dump_json(indent=2))

if __name__ == "__main__":
    main()
