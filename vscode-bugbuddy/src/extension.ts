import * as vscode from 'vscode';
import { initializeServer, disconnectServer } from './client';

export function activate(context: vscode.ExtensionContext) {
	initializeServer().start()
		.then(() => {
			vscode.window.setStatusBarMessage('BugBuddy LSP is ready.', 3000);
		})
		.catch(() => {
			vscode.window.setStatusBarMessage('BugBuddy LSP failed to initialize.', 3000);
		});
}

export async function deactivate() {
	await disconnectServer();
}
