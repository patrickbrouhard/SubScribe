# SubScribe

Your personal scribe automating all your **Obsidian × YouTube** workflows — written in Go.

**SubScribe** fetches metadata and subtitles from YouTube videos and generates structured files to help with knowledge management: transcripts and Markdown notes ready for **Obsidian**.

It also offers **AI workflow assistance**: the program prepares a ready-to-paste prompt for your favorite AI chat, waits for you to get the answer, then embeds it directly into the Obsidian note.

Templates for Obsidian notes and AI prompts are fully editable (see **Templates**).

---

## Table of contents

- [Quickstart](#quickstart)
- [Dependencies](#dependencies)
- [Configuration](#configuration)
- [Configuration resolution order](#configuration-resolution-order)
- [Command-line flags](#command-line-flags)
- [Templates](#templates)
  - [Available fields in `NoteData`](#available-fields-in-notedata)
  - [Available helper functions](#available-helper-functions)
- [Basic usage](#basic-usage)
- [Output structure](#output-structure)

---

## Quickstart

Get started in under a minute 🚀

1. Put `yt-dlp` in the same folder as SubScribe, or set its path in `subscribe.yaml`.

2. Start SubScribe in one of three ways:

   - Run it with a YouTube link in your clipboard.
   - Launch it with no arguments and let it ask for the link.
   - Pass the link directly:

     ```bash
     subscribe --url "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
     ```

SubScribe will extract metadata, download subtitles, build a transcript, copy an AI prompt to your clipboard, and create an Obsidian-ready note.

---

⚠️ **Keep yt-dlp updated:**  
YouTube changes often, and `yt-dlp` needs to stay current.  
If it’s outdated, SubScribe will detect it on startup and show you the latest download link.

---

## Dependencies

You need [`yt-dlp`](https://github.com/yt-dlp/yt-dlp).
Download the appropriate binary for your platform and either place it next to the SubScribe executable, or ensure its path is written in the `subscribe.yaml` config file.

---

## Configuration

`subscribe.yaml` controls how SubScribe behaves. If the file is missing, it is automatically created from the embedded example.

```yaml
# --- Output paths ---
output_dir: "." # Where to store generated files (JSON, transcripts, notes)
obsidian_output_dir: "" # Optional Obsidian vault path (empty = same as output_dir)

# --- Organization ---
save_in_subdir: true # Create a subdirectory per source or project, named after source title

# --- Metadata ---
save_raw_json: false # Save raw metadata JSON from yt-dlp

# --- Subtitles ---
prefer_manual_subs: true # Prefer manual subtitles when available
save_raw_subs: false # Save raw subtitle files (JSON3)

# --- Transcription ---
save_transcript: true # Generate a transcript from subtitles
transcript_format: "txt" # Format of transcript file (txt or md)

# --- Automation ---
auto_mode: false # Run without interaction (can also be set via --auto)

# --- AI Prompt generation ---
generate_ai_prompt: true # Generate an AI-ready prompt from transcript
prompt_split_threshold: 32000 # For information only (splitting not yet implemented)

# --- yt-dlp configuration ---
yt_dlp:
  name: "yt-dlp" # Executable name (".exe" auto-added on Windows)
  path: "" # Directory or absolute path to yt-dlp
  show_warnings: false # Show yt-dlp warnings
  auto_update_check: false # Check for yt-dlp updates automatically

# --- Internal ---
config_version: 1 # Used for config schema migration
```

### Notes

- If `yt_dlp.path` is empty, SubScribe looks for `./yt-dlp(.exe)` first.
- CLI flags (`--auto`, `--config`, etc.) **override** values from the config file.
- The configuration file contains `config_version`; when the schema changes SubScribe will attempt to migrate older files automatically.
- On first run, a default `subscribe.yaml` is created from the embedded example if none exists. Same for the templates.

---

## Configuration resolution order

When SubScribe starts, settings are resolved in this order (highest priority first):

1. **CLI flags** (explicit command-line arguments, e.g. `--auto`, `--url`, `--yt-dlp-path`)
2. **YAML configuration** (`subscribe.yaml`)
3. **Built-in defaults**

So a CLI flag will override the corresponding YAML value for that run only.

---

## Command-line flags

You can override configuration values using CLI flags.

| Flag            | Type   | Description                                                  | Default          |
| --------------- | ------ | ------------------------------------------------------------ | ---------------- |
| `--config`      | string | Path to the YAML config file.                                | `subscribe.yaml` |
| `--url`         | string | YouTube URL to process directly (bypasses manual input).     | _(empty)_        |
| `--auto`        | bool   | Run in automatic mode (no prompts).                          | `false`          |
| `--yt-dlp-path` | string | Absolute path to the `yt-dlp` executable (overrides config). | _(empty)_        |

**Example usage:**

```bash
subscribe --config myconfig.yaml --auto
subscribe --url "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
subscribe --yt-dlp-path "/usr/local/bin/yt-dlp"
```

---

## Templates

SubScribe uses two templates stored in a `templates/` folder next to the binary (and embedded as defaults):

- `templates/obsidian_note.md.tmpl` — defines how the Obsidian note is structured.
- `templates/prompt_for_ai.txt.tmpl` — defines the text prompt used for AI tools.

On startup SubScribe checks the `templates/` folder; if a template is missing it will be copied from the embedded defaults. You can freely edit those files to customize note and prompt output.

If you use Obsidian, you may include a [YAML frontmatter](https://help.obsidian.md/Advanced+topics/YAML+front+matter) block at the top of your note to define metadata (title, tags, aliases, etc.). Frontmatter is optional, but if used make sure it is valid YAML so Obsidian can index your notes properly.

The AI prompt template is plain text and is concatenated with the transcript. The Obsidian note template uses Go’s `text/template` and a `FuncMap` of helper functions (listed below).

---

### Available fields in `NoteData`

The fields below are injected into the `obsidian_note.md.tmpl` template (via `NoteData`):

| Field          | Type        | Description                                       |
| -------------- | ----------- | ------------------------------------------------- |
| `.URL`         | `string`    | Full YouTube video URL.                           |
| `.Title`       | `string`    | Video title.                                      |
| `.Uploader`    | `string`    | Channel / uploader name.                          |
| `.DateStr`     | `string`    | Formatted upload date (e.g. `YYYY-MM-DD`).        |
| `.Categories`  | `[]string`  | Topic categories (if available).                  |
| `.Tags`        | `[]string`  | Tags from metadata.                               |
| `.Hashtags`    | `[]string`  | Hashtags parsed from description.                 |
| `.YtTags`      | `[]string`  | YouTube tags (raw).                               |
| `.Description` | `string`    | Full video description.                           |
| `.Chapters`    | `[]Chapter` | Chapters with timestamp/title/start time.         |
| `.Filename`    | `string`    | Generated filename for the note (safe/sanitized). |
| `.Summary`     | `string`    | AI-generated summary (optional).                  |

> Note: the exact structure and names come from `obsidian.NewNoteData(...)`. If you extend this struct in code, corresponding template fields become available.

---

### Available helper functions

These helpers are available from the `obsidian_note.md.tmpl` template:

| Function                        | Description                                                    |
| ------------------------------- | -------------------------------------------------------------- |
| `yamlList .Tags`                | Formats a string slice as a YAML block list (for frontmatter). |
| `yamlListInline .Tags`          | Formats a string slice as an inline YAML array.                |
| `markdownList .Categories`      | Outputs a Markdown list (`- item`).                            |
| `joinHashtags .Hashtags`        | Joins hashtags with `#` prefixes and spaces.                   |
| `quoteBlock .Description`       | Converts a paragraph into a Markdown quote block.              |
| `formatChapters .Chapters .URL` | Formats YouTube chapters as clickable Markdown links.          |
| `warning "Title" .Text`         | Creates an Obsidian callout of type `[!WARNING]`.              |
| `quote "Author" .Quote`         | Creates an Obsidian callout of type `[!QUOTE]`.                |

---

## Basic usage

```bash
subscribe --url "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
```

By default SubScribe will:

1. **Extract metadata** using `yt-dlp`.
2. **Download subtitles** (manual if available, otherwise automatic).
3. **Generate a transcript** from the subtitles.
4. **Build an AI-ready prompt**, copy it to your clipboard, and wait for your input/answer.
5. **Render an Obsidian-compatible Markdown note** (using the note template) and save output files.

You can run in non-interactive mode with `--auto`. If no `--url` is supplied, SubScribe checks the clipboard and will prompt you if needed.

---

## Output structure

Each processed video creates a directory (when `save_in_subdir: true`) inside `output_dir`:

```
output/
└─ <sanitized-video-title>/
   ├─ metadata.json       # Raw metadata from yt-dlp (if save_raw_json enabled)
   ├─ subtitles.json3     # Raw subtitles file (if save_raw_subs enabled)
   ├─ transcript.txt      # Generated transcript (txt or md)
   ├─ prompt_for_ai.txt   # Full AI prompt text
   └─ obsidian_note.md    # Main Obsidian-compatible note
```

Notes:

- If `obsidian_output_dir` is set, the generated Markdown note is written to that directory (or in addition to `output_dir`, depending on configuration).
- If `save_in_subdir` is `false`, files are written directly into `output_dir`.
- Filenames and directory names are sanitized from the video title to avoid invalid characters.
- Files governed by `save_raw_json` and `save_raw_subs` are omitted when those flags are `false`.

---
