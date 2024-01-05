import { Uri, WorkspaceConfiguration, WorkspaceFolder, window, workspace } from "vscode";

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
	return workspace.getConfiguration('bugbuddy', workspaceFolder.uri);
}
