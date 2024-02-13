import { spawn } from "child_process";
import { platform } from "os";
import { StatusBarAlignment, ThemeColor, Uri, WorkspaceConfiguration, WorkspaceFolder, window, workspace } from "vscode";

const shortExtensionId = 'bugbuddy';
export let extensionId = '';

export function setExtensionId(id: string) {
	extensionId = id;
}

export function getWorkspaceFolder(uri?: Uri): WorkspaceFolder {
	if (uri) {
		return workspace.getWorkspaceFolder(uri)!;
	} else if (window.activeTextEditor && window.activeTextEditor.document) {
		return workspace.getWorkspaceFolder(window.activeTextEditor.document.uri)!;
	} else {
		return workspace.workspaceFolders![0];
	}
}

export function getWorkspaceConfig(): WorkspaceConfiguration {
	const workspaceFolder = getWorkspaceFolder();
	return workspace.getConfiguration(shortExtensionId, workspaceFolder.uri);
}

// Logging
export const outputChannel = window.createOutputChannel('BugBuddy');

export function logErrorString(message: string) {
	window.showErrorMessage(message);
	outputChannel.appendLine(`[BugBuddy - ERROR] ${message}`);
}

export function logError(err: unknown) {
	if (err instanceof Error) {
		logErrorString(err.message);
	} else {
		logErrorString(`Something went wrong: ${err}`);
	}
}

// Status Bar
export enum ConnectionStatus {
	disconnected,
	connecting,
	connected,
	failed,
}

function connStatusToText(status: ConnectionStatus): string {
	switch (status) {
		case ConnectionStatus.disconnected:
			return 'Disconnected';
		case ConnectionStatus.connecting:
			return 'Connecting...';
		case ConnectionStatus.connected:
			return 'Connected';
		case ConnectionStatus.failed:
			return 'Connection Failed';
	}
}

function connStatusToIcon(status: ConnectionStatus): string {
	switch (status) {
		case ConnectionStatus.disconnected:
			return '$(circle-slash)';
		case ConnectionStatus.connecting:
			return '$(sync~spin)';
		case ConnectionStatus.connected:
			return '$(pass-filled)';
		case ConnectionStatus.failed:
			return '$(error)';
	}
}

export const statusBar = window.createStatusBarItem('bugbuddy', StatusBarAlignment.Right, 1000);

export function setConnectionStatus(status: ConnectionStatus, opts?: { participantId: string | null }) {
	const icon = connStatusToIcon(status);
	let text = connStatusToText(status);
	if (status === ConnectionStatus.connected && opts?.participantId) {
		text += ` (${opts.participantId})`;
	}
	statusBar.text = `${icon} BugBuddy: ${text}`;
	if (status === ConnectionStatus.failed) {
		statusBar.backgroundColor = new ThemeColor('errorForeground');
	} else {
		statusBar.backgroundColor = undefined;
	}
}

export function initializeStatusBar() {
	statusBar.name = 'BugBuddy';
	statusBar.tooltip = 'Click to show BugBuddy server menu';
	statusBar.command = 'bugbuddy.showServerMenu';
	setConnectionStatus(ConnectionStatus.disconnected);
	statusBar.show();
}

export function openExplorerIn(path: string, callback: (err: Error) => void) {
	var cmd = ``;
	switch (platform().toLowerCase().replace(/[0-9]/g, ``).replace(`darwin`, `macos`)) {
		case `win`:
			path = path || '=';
			cmd = `explorer`;
			break;
		case `linux`:
			path = path || '/';
			cmd = `xdg-open`;
			break;
		case `macos`:
			path = path || '/';
			cmd = `open`;
			break;
	}
	let p = spawn(cmd, [path]);
	p.on('error', (err) => {
		p.kill();
		return callback(err);
	});
}
