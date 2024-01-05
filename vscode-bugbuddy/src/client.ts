import { ChildProcessWithoutNullStreams, spawn } from "child_process";
import { Uri, ViewColumn, commands, env, window, workspace } from "vscode";
import { LanguageClient, LanguageClientOptions, ServerOptions, State } from "vscode-languageclient/node";
import { extensionId, getWorkspaceConfig, outputChannel } from "./utils";

let serverProcess: ChildProcessWithoutNullStreams;
let client: LanguageClient;

export function getClient(): LanguageClient {
    if (!client) {
        throw new Error('BugBuddy LSP client not initialized.');
    }
    return client;
}

export function launchServer(execPath: string) {
	return spawn(execPath, ["lsp"], {shell: true});
}

export function initializeServer() {
    // register a URI handler for the `openError` command
    // this will open the error in a markdown preview on the side
    window.registerUriHandler({
        async handleUri(uri) {
            if (uri.path !== '/openError') {
                return;
            }

            const params = new URLSearchParams(uri.query);
            const expFile = params.get('file');
            if (!expFile) {
                return;
            }

            const expFileUri = Uri.file(decodeURIComponent(expFile.replace(/\+/g, '%20')));
            await commands.executeCommand('markdown.showPreviewToSide', expFileUri, { locked: true });
        },
    });

    const customPath = getWorkspaceConfig().get<string>('path', 'bugbuddy');
	console.log('Launching bug buddy from', customPath);

	serverProcess = launchServer(customPath);

	serverProcess.stderr.on('data', (raw: string | Buffer) => {
		if (raw instanceof Buffer) {
			outputChannel.appendLine(raw.toString('utf-8'));
		} else {
			outputChannel.appendLine(raw);
		}
	});

	const serverOpts: ServerOptions = () => Promise.resolve(serverProcess);
	const clientOpts: LanguageClientOptions = {
		documentSelector: [{ scheme: 'file' }]
	};

	client = new LanguageClient('BugBuddy LSP', serverOpts, clientOpts);
    return client;
}

function disposeErrorSection() {
    const activeTextEditor = window.visibleTextEditors.find(e => e.document.uri.scheme === extensionId);
    if (!activeTextEditor) {
        return;
    }

    // check if there are editors on beside view column before closing
    // let besideCount = window.visibleTextEditors.filter(e => e.viewColumn === ViewColumn.Beside);
    window.showTextDocument(activeTextEditor.document, activeTextEditor.viewColumn, true);
    commands.executeCommand('workbench.action.closeActiveEditor');
}

export async function disconnectServer() {
    disposeErrorSection();
    await client.stop();
    await client.dispose();
}
