import { expect, test } from "bun:test";
import { harness, tick } from "./text_editor_harness";

test("Save and Save as fall back only above the UTF-8 name bound", async () => {
  for (const name of ["\uD55C".repeat(78) + ".txt", "\uD55C".repeat(80) + ".txt",
    "\u{1F642}".repeat(59) + ".txt", "\u{1F642}".repeat(60) + ".txt",
    "a".repeat(236) + ".txt", "a".repeat(237) + ".txt"]) {
    const ui = await harness();
    ui.setOpen({ cancelled: false, name, text: "opened" });
    await ui.click("open");
    ui.edit("unsaved text");
    ui.setSave({ cancelled: true });
    const suggested = new TextEncoder().encode(name).length <= 240 ? name : "Untitled.txt";
    for (const button of ["save", "save-as"]) {
      await ui.click(button);
      expect(ui.calls.at(-1)).toEqual({ method: "as", params: { text: "unsaved text", name: suggested } });
      expect(ui.node("#document-name").textContent).toBe(name);
      expect(ui.node("#editor").value).toBe("unsaved text");
      expect(ui.node("#save-state").dataset.dirty).toBe("true");
    }
    ui.setFailure(Object.assign(new Error("blocked"), { code: "PERMISSION_DENIED" }));
    await ui.click("save-as");
    expect(ui.node("#document-name").textContent).toBe(name);
    expect(ui.node("#save-state").dataset.dirty).toBe("true");
    ui.setFailure(null);
    ui.setSave({ cancelled: false, name: "chosen.txt", target: 7 });
    await ui.click("save");
    expect(ui.node("#document-name").textContent).toBe("chosen.txt");
    expect(ui.node("#save-state").dataset.dirty).toBe("false");
    ui.edit("next edit");
    await ui.click("save");
    expect(ui.calls.at(-1)).toEqual({ method: "to", params: { text: "next edit", target: 7 } });
  }
});

test("restored long-name draft keeps its name and content until save success", async () => {
  const name = "\uD55C".repeat(80) + ".txt";
  const ui = await harness(true, { draft: { name, text: "restored\r\ntext\r\n" } });
  ui.node("#recovery-dialog").close("recover"); await tick();
  await ui.flushDraft();
  ui.setSave({ cancelled: true });
  await ui.click("save");
  expect(ui.calls.at(-1).params).toEqual({ text: "restored\r\ntext\r\n", name: "Untitled.txt" });
  expect(ui.node("#document-name").textContent).toBe(name);
  expect(ui.storedDraft().name).toBe(name);
  expect(ui.storedDraft().text).toBe("restored\r\ntext\r\n");
  ui.setSave({ cancelled: false, name: "recovered.txt", target: 3 });
  await ui.click("save-as");
  expect(ui.node("#document-name").textContent).toBe("recovered.txt");
  expect(ui.storedDraft()).toBeNull();
});
