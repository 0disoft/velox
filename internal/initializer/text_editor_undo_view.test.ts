import { expect, test } from "bun:test";
import { harness, tick } from "./text_editor_harness";

for (const horizontal of [137, -311]) {
  for (const keyboard of [false, true]) {
    test(`replacement undo retains backward selection and horizontal ${horizontal} via ${keyboard ? "keyboard" : "button"}`, async () => {
      const ui = await harness();
      const text = "x".repeat(220) + "\nSecond line";
      ui.edit(text);
      await ui.click("save");
      await ui.click("replace");
      const input = ui.node("#find-input"), editor = ui.node("#editor");
      input.value = "x"; input.listeners.input();
      ui.node("#replace-input").value = "";
      editor.setSelectionRange(40, 46, "backward");
      editor.scrollTop = 53;
      editor.scrollLeft = horizontal;
      ui.node("#replace-all").click();
      expect(editor.value).toBe("\nSecond line");
      editor.scrollTop = editor.scrollLeft = 0;
      if (keyboard) {
        ui.node("#find-close").click();
        let prevented = false;
        ui.listeners.keydown({ ctrlKey: true, code: "KeyZ", target: editor, preventDefault() { prevented = true; } });
        expect(prevented).toBe(true);
      } else ui.node("#replace-undo").click();
      await tick();
      expect(editor.value).toBe(text);
      expect([editor.selectionStart, editor.selectionEnd, editor.selectionDirection, editor.scrollTop, editor.scrollLeft])
        .toEqual([40, 46, "backward", 53, horizontal]);
      expect(ui.document.activeElement).toBe(editor);
      expect(ui.node("#save-state").dataset.dirty).toBe("false");
      expect(ui.calls).toHaveLength(1);
      await ui.flushDraft();
      expect(ui.storedDraft()).toBeNull();
      expect(ui.node("#replace-undo").disabled).toBe(true);
    });
  }
}
