# File Notes

File Notes is a local Markdown and text editor built entirely on browser-owned
APIs exposed by the Velox WebView2 origin. It uses the File System Access API
for explicit open and save gestures and IndexedDB for one recoverable draft.

The application has no native permission, network request, frontend package,
bundler, or generated binding. It accepts only files selected by the user,
rejects files larger than 2 MiB, never stores a filesystem path, and asks before
discarding unsaved changes.

On supported Velox hosts, normal native window close uses the browser's
`beforeunload` confirmation. Cancel keeps the current document open even when
draft recovery is unavailable. Forced process termination is not protected by
that confirmation, and recovery still requires a completed IndexedDB write.

Browser support does not guarantee that Windows policy or a particular WebView2
runtime grants every operation. Unsupported or denied picker operations remain
visible application states rather than falling back to unrestricted native
filesystem access.
