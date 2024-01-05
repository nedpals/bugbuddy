import { ChildProcessWithoutNullStreams, spawn } from "child_process";
import { Uri, ViewColumn, commands, window } from "vscode";
import { LanguageClient, LanguageClientOptions, ServerOptions, State } from "vscode-languageclient/node";
import { getWorkspaceConfig } from "./utils";

let serverProcess: ChildProcessWithoutNullStreams;
let client: LanguageClient;

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
			console.error(raw.toString('utf-8'));
		} else {
			console.error(raw);
		}
	});

	const serverOpts: ServerOptions = () => Promise.resolve(serverProcess);
	const clientOpts: LanguageClientOptions = {
		documentSelector: [{ scheme: 'file' }]
	};

	client = new LanguageClient('BugBuddy LSP', serverOpts, clientOpts);

	client.onDidChangeState(event => {
		if (event.newState === State.Running) {
			client.onNotification('textDocument/publishDiagnostic', (req) => {
				const view = window.createWebviewPanel(
					'bugbuddyError',
					'BugBuddy',
					ViewColumn.Active,
					{}
				);

				view.webview.html = '<h1>Hello BugBuddy</h1>';
			});
		}
	});

    return client;
}

export async function disconnectServer() {
    await client.stop();
    await client.dispose();
}
