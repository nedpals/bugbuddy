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

export function activate(context: vscode.ExtensionContext) {
	// vscode.window.showInformationMessage("Launching BugBuddy...");

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
