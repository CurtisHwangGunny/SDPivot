# SDPivot User Manual

This manual explains the day-to-day SDPivot user workflow: signing in, organizing knowledge spaces, importing and managing documents, asking AI questions, creating AI-assisted drafts, finding content, reporting problems, and managing Q&A sessions.

The instructions reflect the user interface currently implemented in this repository. Some controls are shown only when your account has the required role, and some backend capabilities are not yet exposed as buttons in the current UI.

## 1. Before you begin

You need:

- The URL of your SDPivot deployment.
- A registered account or permission to register a new account.
- Access to at least one tenant and, for private spaces, membership in the relevant knowledge space.
- An active AI model configured by an administrator before using AI Q&A or AI Writing.

After authentication, SDPivot opens **Knowledge Spaces** at `/spaces`. The main navigation contains:

| Menu | Purpose |
| --- | --- |
| **Knowledge Spaces** | Create spaces, view space details, import documents, and manage members |
| **AI Q&A** | Create conversations and ask questions against indexed knowledge |
| **AI Writing** | Create, generate, edit, save, and export drafts |
| **Administration** | Tenant administration, subject to your permissions |
| **Enterprise Management** | Enterprise and organization functions, subject to your permissions |
| **Usage Statistics** | View available usage information |
| **Personal Settings** | Manage your profile, preferences, and appearance |

The sidebar also provides **Light**, **Dark**, and **System** theme options.

## 2. Login and registration

### 2.1 Sign in with a phone number

1. Open `/login`.
2. Select **Phone Login**.
3. Enter your phone number and password.
4. Review and accept the service agreement and privacy policy checkbox.
5. Select **Login**.

Phone input accepts digits only and is limited to 11 digits. A successful login opens **Knowledge Spaces**.

### 2.2 Sign in with email

1. Open `/login`.
2. Select **Email Login**.
3. Enter your email address and password.
4. Accept the service agreement and privacy policy.
5. Select **Login**.

The **WeChat QR Code** tab is currently disabled and cannot be used to sign in.

### 2.3 Register an account

1. On the login page, select **Register Now**, or open `/register`.
2. Enter a phone number.
3. Optionally enter a display name.
4. Enter a password of at least eight characters.
5. Enter the same password again in **Confirm Password**.
6. Accept the service agreement and privacy policy.
7. Select **Register**.

Registration signs you in automatically and opens `/spaces`. If registration is restricted by deployment policy, contact an administrator for account provisioning.

### 2.4 Sign out

1. Open the user menu at the bottom of the sidebar.
2. Select **Logout**.

SDPivot clears the local authentication state and returns to `/login`.

## 3. Knowledge spaces

A knowledge space groups related documents, collaborators, Q&A context, and writing sources. A useful structure is one space per department, project, product, or business topic.

### 3.1 Browse spaces

Open **Knowledge Spaces** or `/spaces`. Each row shows the space name, description, visibility, and ID.

Visibility labels are:

| Visibility | Meaning |
| --- | --- |
| **Private** | Restricted to explicitly authorized users |
| **Team visible** | Available according to team access rules |
| **Enterprise visible** | Available according to enterprise access rules |

Select a space row or **Enter** to open its details. Select **Refresh List** to reload the page data.

### 3.2 Create a space

1. Open **Knowledge Spaces**.
2. Select **New Space**.
3. Enter a clear space name.
4. Add a description that explains what belongs in the space.
5. Choose the required visibility.
6. Confirm creation.

Use names that make the intended scope obvious, such as `Finance Policies`, `Product Alpha`, or `Customer Support`.

### 3.3 Review space status

The space detail page, `/spaces/<space-id>`, shows:

- Space description and visibility.
- Creation date.
- Total documents.
- Successfully parsed documents.
- Documents waiting for or undergoing parsing.
- Member count.
- A document summary table.

Only documents whose parsing status is **Completed** are ready to contribute reliably to search and AI answers.

### 3.4 Manage space members

If your role permits member administration:

1. Open the space.
2. Select **Member Management**.
3. Enter the member's user ID.
4. Select a role.
5. Select **Add Member**.

Available space roles are:

| Role | Intended use |
| --- | --- |
| **Viewer** | Read and use accessible knowledge |
| **Editor** | Maintain space content in addition to reading it |
| **Administrator** | Manage the space and its membership |

To remove access, select **Remove** beside a member. If a member action is rejected, verify that you have the required space or tenant role and that the user ID is correct.

## 4. Document upload and management

### 4.1 Open document management

From a space detail page, select **Document Management**, or open:

```text
/spaces/<space-id>/documents
```

The page lists document name, type, size, parse status, chunk count, upload date, and available actions.

### 4.2 Upload files

1. Open a knowledge space.
2. Select **Import Document**.
3. Choose **File Upload**.
4. Select files or drag them into the upload area.
5. Wait for the upload confirmation.

The full document-management dialog accepts up to 20 files at a time, with a maximum of 50 MB per file. Supported selections include:

- PDF
- Word: `.doc`, `.docx`
- Excel: `.xls`, `.xlsx`
- PowerPoint: `.ppt`, `.pptx`
- Markdown: `.md`
- Plain text: `.txt`
- Images: `.png`, `.jpg`, `.jpeg`
- Audio: `.mp3`

Successful upload does not necessarily mean the document is immediately searchable. Check its parse status until it becomes **Completed**.

### 4.3 Import a webpage

The webpage option is available from the document-management import dialog.

1. Open **Document Management** for a space.
2. Select **Import Document**.
3. Select **Web Link**.
4. Enter a complete `http://` or `https://` URL.
5. Optionally enter comma-separated tags.
6. Select **Import Webpage**.

SDPivot fetches the page's main text and creates a document in the selected space. The deployment's network and security policy may reject inaccessible or disallowed destinations.

### 4.4 Create a document manually

1. Open **Document Management** and select **Import Document**.
2. Select **Manual Entry**.
3. Enter a required title.
4. Enter the content in Markdown format.
5. Optionally enter comma-separated tags.
6. Select **Save**.

Manual entry is useful for notes, policies, meeting summaries, and content copied from systems that do not support direct export.

### 4.5 Understand parse status

| Status | Meaning | Recommended action |
| --- | --- | --- |
| **Pending** | The document is waiting for processing | Wait and refresh later |
| **Parsing** | Content is being extracted and split into chunks | Do not repeatedly trigger reparse |
| **Completed** | Parsing finished and chunks are available | The document is ready for retrieval |
| **Failed** | Parsing did not complete | Check the file, then select **Reparse** |

### 4.6 Inspect document chunks

Select a document row or select **Chunks**. The chunk dialog displays the indexed sections and their order. Use this view to check whether:

- The expected text was extracted.
- Headings and paragraphs were split sensibly.
- Scanned or image-heavy files produced usable text.
- The answer source exists in the indexed content.

### 4.7 Reparse a document

Select **Reparse** beside the document. Use this after a failed parse or when the document needs to be processed again. The status returns to a processing state while new chunks are created.

### 4.8 Delete a document

1. Select **Delete** beside the document.
2. Confirm the prompt.

Deletion removes the document from the active document list and from future retrieval. Treat it as destructive unless your deployment has an external backup or retention process.

## 5. AI Q&A

Open **AI Q&A** or `/qa`.

### 5.1 Start a conversation

1. Select **New Session** or **Start Now**.
2. Enter a question in the input area.
3. Select an available answer model.
4. Select **Send Question**.

The Send button remains unavailable until both a non-empty question and a model are selected. If no models appear, an administrator must activate a Q&A-capable model.

### 5.2 Ask effective questions

For better retrieval and answers:

- Name the topic, product, department, or policy explicitly.
- Ask one main question at a time.
- Include relevant dates, version numbers, or document names.
- Request a specific output format when useful, such as a checklist, table, or summary.
- Ask the assistant to identify uncertainty when the source material is incomplete.

Example:

```text
Summarize the approval requirements in the 2026 travel policy. Return a table with expense type, approver, and required evidence.
```

### 5.3 Review an answer and its sources

The conversation displays user and AI messages in chronological order. When source information is available, it appears beneath the AI response.

Before relying on an answer:

1. Read the cited source information.
2. Confirm the relevant document is in **Completed** status.
3. Compare important dates, amounts, names, and requirements with the source document.
4. Ask a follow-up question if the response is incomplete or ambiguous.

AI output can be inaccurate. Do not use it as the sole basis for legal, medical, financial, safety, or other high-impact decisions.

### 5.4 Continue an existing conversation

Select a session in the left panel. SDPivot reloads its messages so you can continue with the existing context. Session titles and update dates help identify recent conversations.

## 6. AI Writing

Open **AI Writing** or `/writing`.

### 6.1 Create a draft

1. Select **New Draft**.
2. Enter a draft title.
3. Confirm creation.
4. Select the draft from the left panel if it is not already active.

The draft editor supports direct editing in addition to AI generation.

### 6.2 Generate content

1. Enter instructions, an outline, or source text in the editor. If the editor is empty, the draft title is used as the prompt.
2. Choose a source mode:
   - **Knowledge Base Only** prioritizes indexed internal knowledge.
   - **Knowledge Base + Internet** supplements internal knowledge with public web information.
3. Select **AI Generate**.
4. Review the generated text.

After generation, SDPivot shows the number of knowledge and web sources used and the selected model. Generation replaces the editor content with the returned result and then saves the draft.

### 6.3 Edit and save

- Edit the title in the header.
- Edit the body in the main text area.
- Select **Save**, or leave a title/content field to trigger the current blur-save behavior.
- Check the draft's updated date in the left panel after the draft list refreshes.

Always review generated content for correctness, confidentiality, tone, and unsupported claims.

### 6.4 Export a draft

Select **Export**. The current UI downloads the draft as a `.docx` file named from the draft title.

If export fails, save the draft first, retry once, and then contact support with the draft title and approximate failure time.

## 7. Tags: browsing and filtering

Tags can be attached as comma-separated text when importing a webpage or creating a document manually. Use consistent terms such as `finance, travel, 2026` rather than creating spelling variants.

### Current UI behavior

- The SDPivot document record stores tags.
- The import dialog accepts tags for webpage and manual imports.
- The current SDPivot document table does not display a dedicated tag column.
- The current SDPivot document page does not provide a tag browser or tag-filter control.
- The document list API currently filters by title search and parse status, not by tag.

Therefore, tag browsing and filtering cannot currently be completed from the SDPivot user interface. Use title conventions and separate knowledge spaces as the practical organization method until a tag filter is mounted. Do not assume entering a tag in the document-name search box will filter by tag.

## 8. Search

### 8.1 Search document titles

1. Open a space's **Document Management** page.
2. Enter text in **Search Documents**.
3. Press Enter.

This searches document titles in the current space. The search is not a tag filter and does not search full document content.

### 8.2 Filter by parse status

Use the **Status** selector to show documents in one of these states:

- Pending
- Parsing
- Completed
- Failed

Clear the selector to return to all statuses.

### 8.3 Search document content

SDPivot has a protected semantic/content search API, but the current SDPivot navigation does not mount a standalone search page or search-results component. For normal users, use **AI Q&A** to retrieve information from completed documents.

When a known document is not found:

1. Confirm you are in the correct knowledge space.
2. Clear the status filter.
3. Search using part of the document title.
4. Confirm the document has not been deleted.
5. Confirm your account still has access to the space.

## 9. Feedback and problem reporting

### 9.1 Review feedback shown by the application

SDPivot displays success and error messages after actions such as login, upload, import, reparse, deletion, save, generation, and export. Read the message before retrying; repeated actions can create duplicate uploads or unnecessary processing jobs.

### 9.2 Report incorrect AI output

The current SDPivot Q&A interface does not provide thumbs-up, thumbs-down, rating, or free-text feedback controls. To report an incorrect answer to your support or administrator team, include:

- Your tenant or enterprise name.
- The Q&A session title.
- The exact question.
- The incorrect or incomplete answer excerpt.
- The source label shown beneath the answer, if present.
- The expected answer and supporting document name.
- The approximate time of the request.

Do not include passwords, access tokens, or unrelated confidential information.

### 9.3 Report application errors

For upload, parsing, search, Q&A, writing, or export failures, record:

- The page and action that failed.
- The exact on-screen error message.
- The knowledge space and document or draft title.
- File type and size for upload problems.
- Whether retrying once produced the same result.
- Browser and approximate timestamp.

The current user UI does not contain a dedicated feedback form. Follow your organization's support channel or contact a tenant administrator.

## 10. Session management

### 10.1 Create and switch sessions

- Select **New Session** to start a separate topic.
- Select a session in the left panel to reopen its conversation.
- Sessions are listed with the most recently updated conversations first.
- The service returns up to 50 sessions in the current list view.

Create a new session when changing topics substantially. This keeps context focused and makes past work easier to find.

### 10.2 Session titles

New sessions receive a timestamp-based title such as `New Session 14:30`. The current Q&A UI does not provide a rename control, so use one session per topic and rely on its creation/update time to distinguish it.

### 10.3 Delete sessions

The backend supports deleting a session, including its messages, but the current SDPivot Q&A page does not expose a delete button. Session deletion is therefore not available through the current user interface.

Until a delete control is added:

- Avoid placing unnecessary secrets or personal data in prompts.
- Start a new session for unrelated work.
- Contact an administrator or authorized API operator if deletion is required by policy.

### 10.4 Session privacy

Q&A sessions are scoped to the authenticated user and tenant. A session associated with a restricted space also requires continuing access to that space. Sign out when using a shared computer.

## 11. Troubleshooting

| Problem | What to check |
| --- | --- |
| Login fails | Verify the selected phone/email tab, credentials, agreement checkbox, and account status |
| No spaces are visible | Confirm tenant membership and space access with an administrator |
| Upload fails | Check format, 50 MB file limit, network connection, and authentication session |
| Document remains pending | Wait and refresh; if it does not progress, ask an administrator to check processing services |
| Parsing fails | Inspect the source file, then use **Reparse**; image/scanned content may require additional parser support |
| Search returns nothing | Check the current space, title spelling, and status filter |
| Q&A model selector is empty | Ask an administrator to activate a Q&A model |
| AI answer misses a document | Confirm the document is in the correct space and has **Completed** status |
| AI generation fails | Save the draft, confirm model availability, and retry once |
| Export fails | Save the draft first, retry, then report the draft title and timestamp |
| A control is missing | Your role may not permit it, or the backend capability may not yet be mounted in the current UI |

## 12. Safe-use checklist

- Verify important AI answers against source documents.
- Keep confidential data within spaces and roles approved for that data.
- Use **Knowledge Base Only** for writing when public web information is not appropriate.
- Review generated drafts before distribution.
- Delete obsolete documents when policy permits and an authoritative replacement exists.
- Sign out on shared devices.
- Never send passwords, access tokens, or private keys through Q&A, writing prompts, or support reports.
