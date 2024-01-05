import * as vscode from 'vscode';
import { initializeServer, disconnectServer } from './client';
import { disposeTerminal, runFromUri } from './runner';
import { setExtensionId } from './utils';

export function activate(context: vscode.ExtensionContext) {
	setExtensionId(context.extension.id);

	context.subscriptions.push(
		vscode.commands.registerCommand('bugbuddy.run', runFromUri)
	);

	initializeServer().start()
		.then(() => {
			vscode.window.setStatusBarMessage('BugBuddy LSP is ready.', 3000);
		})
		.catch(() => {
			vscode.window.setStatusBarMessage('BugBuddy LSP failed to initialize.', 3000);
		});
}

export async function deactivate() {
	await disposeTerminal();
	await disconnectServer();
}
