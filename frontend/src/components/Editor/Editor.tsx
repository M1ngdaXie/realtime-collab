import Collaboration from "@tiptap/extension-collaboration";
import CollaborationCursor from "@tiptap/extension-collaboration-cursor";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { useEffect } from "react";
import type { YjsProviders } from "../../lib/yjs";
import Toolbar from "./Toolbar.tsx";

interface EditorProps {
  providers: YjsProviders;
  permission: string | null;
}

const Editor = ({ providers, permission }: EditorProps) => {
  // Default to false (read-only) until permission is loaded
  const isEditable = permission === "edit";

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        history: false, // Use Yjs history instead
      }),
      Collaboration.configure({
        document: providers.ydoc,
      }),
      CollaborationCursor.configure({
        provider: providers.websocketProvider,
        user: providers.awareness.getLocalState()?.user || {
          name: "Anonymous",
          color: "#000000",
        },
      }),
    ],
    content: "",
    editable: isEditable,
  });

  // Update editable state when permission changes
  useEffect(() => {
    if (editor) {
      editor.setEditable(permission === "edit");
    }
  }, [editor, permission]);

  useEffect(() => {
    if (editor && providers.awareness) {
      const user = providers.awareness.getLocalState()?.user;
      if (user) {
        editor.extensionManager.extensions.forEach((extension: any) => {
          if (extension.name === "collaborationCursor") {
            extension.options.user = user;
          }
        });
      }
    }
  }, [editor, providers.awareness]);

  if (!editor) {
    return <div>Loading editor...</div>;
  }

  return (
    <div className="editor-container">
      {isEditable && <Toolbar editor={editor} />}
      <EditorContent
        editor={editor}
        className={`editor-content ${!isEditable ? 'view-only' : ''}`}
      />
    </div>
  );
};

export default Editor;
