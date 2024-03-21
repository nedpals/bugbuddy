import { Terminal, TextDocument, Uri, window, workspace } from "vscode";
import { getClient } from "./client";
import { logError } from "./utils";

export let terminal: Terminal | null;

export async function runFromUri(uri?: Uri): Promise<void> {
    if (!uri) {
        if (!window.activeTextEditor) {
            throw new Error('No file to run.');
        }
        return runFromUri(window.activeTextEditor.document.uri);
    }

    try {
        const doc = await workspace.openTextDocument(uri);
        await runDocument(doc);
    } catch (e) {
        logError(e);
    }
}

export async function runDocument(doc: TextDocument) {
    if (doc.isDirty || doc.isUntitled) {
        throw new Error('Please save the file before running.');
    }

    const { command } = await getClient().sendRequest<{ command: string }>('$/fetchRunCommand', {
        languageId: doc.languageId,
        textDocument: {
            uri: doc.uri.toString(),
        }
    });

    if (!command) {
        throw new Error('Cannot run this file.');
    }

    if (!terminal) {
        // look for existing terminal before creating a new one
        // this is to prevent creating multiple terminals
        window.terminals.forEach((t) => {
            if (t.name === 'BugBuddy' && t.exitStatus === undefined) {
                terminal = t;
            }
        });

        // if no terminal found, create a new one
        if (!terminal) {
            terminal = window.createTerminal('BugBuddy');
        }
    }

    terminal.show();
    terminal.sendText(command);
}

export async function disposeTerminal() {
    if (!terminal) {
        return;
    }
    terminal.dispose();
    terminal = null;
}
