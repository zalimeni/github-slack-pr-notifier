#!/usr/bin/env python3
import argparse
import json
import sys
from pathlib import Path


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate a customized Slack workflow JSON export from the checked-in template."
    )
    parser.add_argument(
        "--template",
        default="docs/workflow-example.json",
        help="Path to the source workflow JSON template.",
    )
    parser.add_argument(
        "--title",
        default="GitHub Webhook - Notifications",
        help="Workflow title to set in the export.",
    )
    parser.add_argument(
        "--description",
        default="Adapted from exported workflow for GitHub PR notifications",
        help="Workflow description to set in the export.",
    )
    parser.add_argument(
        "--icon-url",
        default="https://raw.githubusercontent.com/zalimeni/github-slack-pr-notifier/main/docs/assets/workflow-icon.png",
        help="Hosted icon URL to embed in the export.",
    )
    parser.add_argument(
        "--channel-id",
        default=None,
        help="Slack channel ID to write into all send-message steps.",
    )
    return parser.parse_args()


def load_template(path: Path) -> dict:
    with path.open("r", encoding="utf-8") as handle:
        return json.load(handle)


def rewrite_channel_ids(node: object, channel_id: str) -> None:
    if isinstance(node, dict):
        if set(node.keys()) >= {"channel_id"}:
            channel = node.get("channel_id")
            if isinstance(channel, dict) and "value" in channel:
                channel["value"] = channel_id
        for value in node.values():
            rewrite_channel_ids(value, channel_id)
        return

    if isinstance(node, list):
        for value in node:
            rewrite_channel_ids(value, channel_id)


def main() -> int:
    args = parse_args()
    workflow_doc = load_template(Path(args.template))

    workflow = workflow_doc.get("workflow")
    triggers = workflow_doc.get("triggers")
    if not isinstance(workflow, dict) or not isinstance(triggers, list):
        raise SystemExit("unexpected workflow JSON shape")

    workflow["title"] = args.title
    workflow["description"] = args.description
    workflow["icon"] = args.icon_url

    for trigger in triggers:
        if isinstance(trigger, dict):
            trigger["name"] = args.title

    if args.channel_id:
        rewrite_channel_ids(workflow, args.channel_id)

    json.dump(workflow_doc, sys.stdout, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
