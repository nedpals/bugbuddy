import { Uri, WorkspaceConfiguration, WorkspaceFolder, window, workspace } from "vscode";

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
