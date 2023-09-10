import * as vscode from 'vscode';
import { LanguageClient, LanguageClientOptions, ProtocolRequestType0, ServerOptions, State } from 'vscode-languageclient/node';
import { ChildProcessWithoutNullStreams, spawn } from 'child_process';

function getWorkspaceFolder(uri?: vscode.Uri): vscode.WorkspaceFolder {
	if (uri) {
		return vscode.workspace.getWorkspaceFolder(uri)!;
	} else if (vscode.window.activeTextEditor && vscode.window.activeTextEditor.document) {
		return vscode.workspace.getWorkspaceFolder(vscode.window.activeTextEditor.document.uri)!;
	} else {
		return vscode.workspace.workspaceFolders![0];
	}
}

function getWorkspaceConfig(): vscode.WorkspaceConfiguration {
	const workspaceFolder = getWorkspaceFolder();
	return vscode.workspace.getConfiguration('bugbuddy', workspaceFolder.uri);
}

function launchServer(execPath: string) {
	return spawn(execPath, ["lsp"], {shell: true});
}

let serverProcess: ChildProcessWithoutNullStreams;
let client: LanguageClient;

interface ErrorPayload {
	template: string
	language: string
	message: string
	location: {
		// eslint-disable-next-line @typescript-eslint/naming-convention
		DocumentPath: string
		// eslint-disable-next-line @typescript-eslint/naming-convention
		Position: {
			// eslint-disable-next-line @typescript-eslint/naming-convention
			Line: number
			// eslint-disable-next-line @typescript-eslint/naming-convention
			Column: number
			// eslint-disable-next-line @typescript-eslint/naming-convention
			Index: number
		}
	}
}

export function activate(context: vscode.ExtensionContext) {
	let errorPanel: vscode.WebviewPanel | null;
	let currentPathOpenedForError: string | null;

	vscode.window.registerUriHandler({
		async handleUri(uri) {
			if (uri.path !== '/openError') {
				return;
			}

			const params = new URLSearchParams(uri.query);
			const errorId = params.get('id');
			if (!errorId) {
				return;
			}

			const { template, language, message, location } = await client.sendRequest<ErrorPayload>('$/viewError', { id: parseInt(errorId) });

			if (!currentPathOpenedForError) {
				currentPathOpenedForError = location.DocumentPath;
			}

			if (vscode.window.activeTextEditor?.document.uri.fsPath !== currentPathOpenedForError) {
				const doc = await vscode.workspace.openTextDocument(vscode.Uri.file(currentPathOpenedForError));
				await vscode.window.showTextDocument(doc, vscode.ViewColumn.One, false);
			}

			if (!errorPanel) {
				errorPanel = vscode.window.createWebviewPanel(
					'bugbuddyError',
					'BugBuddy',
					vscode.ViewColumn.Two,
					{}
				);
			}

			errorPanel.webview.html = `
			<h1>${language} / ${template}</h1>
			<p>${message}</p>
			`;

			errorPanel.onDidDispose(() => {
				errorPanel = null;
			}, null, context.subscriptions);
		},
	});

	vscode.workspace.onDidCloseTextDocument(closedDoc => {
		console.log(closedDoc.uri.fsPath, currentPathOpenedForError);
		if (closedDoc.uri.fsPath !== currentPathOpenedForError) {
			return;
		}

		errorPanel?.dispose();
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
				const view = vscode.window.createWebviewPanel(
					'bugbuddyError',
					'BugBuddy',
					vscode.ViewColumn.Active,
					{}
				);

				view.webview.html = '<h1>Hello BugBuddy</h1>';
			});
		}
	});

	client.start()
		.then(() => {
			vscode.window.setStatusBarMessage('BugBuddy LSP is ready.', 3000);
		})
		.catch(() => {
			vscode.window.setStatusBarMessage('BugBuddy LSP failed to initialize.', 3000);
		});
}

export async function deactivate() {
	await client.stop();
	await client.dispose();
}
