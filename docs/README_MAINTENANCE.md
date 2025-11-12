# 📝 README Maintenance Guide

> How to keep the README.md fresh and accurate as the project evolves

---

## 🔄 When to Update README

Update the README.md whenever:

1. **New Features Added**: Update the Features section and Roadmap
2. **Architecture Changes**: Update diagrams and architecture description
3. **New Dependencies**: Update Tech Stack table
4. **API Changes**: Update API Documentation section
5. **Configuration Changes**: Update Quick Start and Configuration sections
6. **Milestone Completed**: Update Roadmap status and completion dates
7. **Coverage Changes**: Update test coverage badges and percentages
8. **New Documentation**: Add links to new docs in relevant sections

---

## 📋 Update Checklist

### When Completing a Roadmap Task

- [ ] Update task status (⏳ → 🔄 → ✅)
- [ ] Add completion date
- [ ] Update features section if new feature
- [ ] Update API documentation if endpoints changed
- [ ] Update test coverage percentage
- [ ] Update badges if applicable
- [ ] Commit with `docs: update README for task #XX completion`

### When Adding New Features

- [ ] Add to Features table
- [ ] Update architecture diagram if needed
- [ ] Document new API endpoints
- [ ] Update configuration examples
- [ ] Add to Quick Start if setup required
- [ ] Update Make targets if new commands
- [ ] Document new environment variables

### When Changing Architecture

- [ ] Update Mermaid diagram in Architecture section
- [ ] Update Project Structure tree
- [ ] Update Architecture Principles if needed
- [ ] Update ARCHITECTURE.md reference
- [ ] Consider adding sequence diagrams

---

## 🎨 Style Guide

### Badge Updates

Badges are at the top of README. Update URLs in this format:

```markdown
[![Badge Name](https://img.shields.io/badge/...)](link)
```

**Common Badges:**
- CI Pipeline: Update workflow name in URL
- Go Version: Automatically pulls from go.mod
- Coverage: Update percentage manually after coverage report
- License: Static, only change if license changes

### Mermaid Diagrams

Update diagrams when architecture changes:

```markdown
```mermaid
graph TB
    A[Component] --> B[Component]
```\`\`\`

**Tips:**
- Keep diagram simple and focused
- Use consistent styling with `style` commands
- Test diagrams render correctly on GitHub
- Consider creating separate diagram files in `docs/assets/`

### Tables

Maintain consistent table formatting:

```markdown
| Column 1 | Column 2 | Column 3 |
|----------|----------|----------|
| Data     | Data     | Data     |
```

**Roadmap Table:**
- Use emoji for status: ✅ 🔄 ⏳ 🔴
- Include all columns: #, Task, Status, Priority, ETA
- Keep dates in format: Nov 12, 2025 or Q1 2025

### Emojis

Consistent emoji usage:

- 🚀 Features, Quick Start
- 🔐 Security
- 📊 Metrics, Observability
- 🧪 Testing
- 📚 Documentation
- 🗺️ Roadmap
- 🤝 Contributing
- ⭐ Support

---

## 🛠️ Common Updates

### 1. Update Test Coverage

```bash
# Run coverage
make test-coverage

# Update README badge
# [![Code Coverage](https://img.shields.io/badge/coverage-XX%25-brightgreen.svg)]

# Update Current Coverage section
```

### 2. Update Roadmap Status

```markdown
| 27 | ✅ Rate Limiting Middleware | **Completed** | P2 | ✓ Nov 12 |
```

### 3. Add New API Endpoint

```markdown
| `POST` | `/api/v1/new-endpoint` | Description | ✅ JWT |
```

### 4. Update Environment Variables

Add to Configuration section:

```bash
# New Feature
NEW_FEATURE_ENABLED=true
NEW_FEATURE_CONFIG=value
```

### 5. Add New Make Target

```markdown
| `make new-command` | Description of command |
```

---

## 📸 Asset Management

### Images & Diagrams

Store in `docs/assets/`:

```
docs/assets/
├── architecture-overview.png
├── deployment-diagram.png
├── sequence-login.png
└── logo.png
```

**Reference in README:**

```markdown
![Architecture](./docs/assets/architecture-overview.png)
```

**Creating Diagrams:**
- Use Mermaid for simple flow diagrams (renders on GitHub)
- Use draw.io or PlantUML for complex diagrams
- Export as PNG with transparent background
- Keep file sizes under 1MB
- Use descriptive filenames

### Logo & Branding

If adding logo:

```markdown
<div align="center">
  <img src="./docs/assets/logo.png" alt="Pandora Exchange" width="200"/>
</div>
```

---

## ✅ Pre-Commit Checklist

Before committing README changes:

- [ ] All links work (no 404s)
- [ ] Mermaid diagrams render correctly
- [ ] Tables formatted consistently
- [ ] Badges point to correct URLs
- [ ] Code blocks have correct syntax highlighting
- [ ] Spelling and grammar checked
- [ ] Section order makes sense
- [ ] Table of contents updated if sections added/removed
- [ ] Version/date updated at bottom
- [ ] Follows ag-ui visual style

---

## 🔗 Quick Links to Update

When updating README, check these related files:

- `ARCHITECTURE.md` - Architecture specification
- `docs/testing.md` - Testing guide
- `docs/security/` - Security documentation
- `docs/observability/` - Observability guide
- `.github/workflows/` - CI/CD workflows
- `Makefile` - Build commands

---

## 📅 Review Schedule

**Monthly:**
- [ ] Verify all links work
- [ ] Update test coverage percentages
- [ ] Review roadmap progress
- [ ] Update completion dates

**Quarterly:**
- [ ] Review entire README for accuracy
- [ ] Update architecture diagrams
- [ ] Refresh screenshots if UI changed
- [ ] Update dependency versions in tech stack

**On Release:**
- [ ] Update version numbers
- [ ] Update completion status for all tasks
- [ ] Add release notes reference
- [ ] Update "Last Updated" date

---

## 🎯 Quality Standards

A good README should:

- ✅ Load in under 3 seconds
- ✅ Have visual hierarchy (headers, tables, diagrams)
- ✅ Be scannable (emojis, bold, tables)
- ✅ Include working code examples
- ✅ Have up-to-date information
- ✅ Link to detailed docs for deep dives
- ✅ Follow ag-ui visual style
- ✅ Be mobile-friendly

---

## 🚫 What NOT to Include

- ❌ Sensitive information (passwords, tokens, API keys)
- ❌ Personal contact information
- ❌ Outdated screenshots or diagrams
- ❌ Broken links
- ❌ Implementation details (belongs in ARCHITECTURE.md)
- ❌ Large images (>1MB)
- ❌ Long code examples (link to docs/ instead)

---

## 🤖 Automation Ideas

Consider automating:

1. **Badge Updates**: CI workflow to update coverage badge
2. **Link Checking**: Pre-commit hook to verify links
3. **Diagram Generation**: Script to generate Mermaid from code
4. **Changelog**: Auto-update roadmap from commits
5. **Table of Contents**: Auto-generate from headers

---

## 📞 Questions?

If unsure about a README change:

1. Check this guide
2. Review [ag-ui README](https://github.com/ag-ui-protocol/ag-ui) for style reference
3. Ask in GitHub Discussions
4. Create draft PR for feedback

---

**Remember:** The README is the first impression of the project. Keep it fresh, accurate, and visually appealing!
