import * as vscode from 'vscode';
import { disconnectServer, generateParticipantId, showServerMenu, startServer } from './client';
import { disposeTerminal, runFromUri } from './runner';
import { initializeStatusBar, setExtensionId } from './utils';

export function activate(context: vscode.ExtensionContext) {
	setExtensionId(context.extension.id);
	context.subscriptions.push(
		vscode.commands.registerCommand('bugbuddy.run', runFromUri),
		vscode.commands.registerCommand('bugbuddy.showServerMenu', showServerMenu),
        vscode.commands.registerCommand('bugbuddy.generateParticipantId', generateParticipantId),
	);

    // register a URI handler for the `openError` command
    // this will open the error in a markdown preview on the side
    vscode.window.registerUriHandler({
        async handleUri(uri) {
            if (uri.path !== '/openError') {
                return;
            }

            const params = new URLSearchParams(uri.query);
            const expFile = params.get('file');
            if (!expFile) {
                return;
            }

            const expFileUri = vscode.Uri.file(decodeURIComponent(expFile.replace(/\+/g, '%20')));
            await vscode.commands.executeCommand('markdown.showPreviewToSide', expFileUri, { locked: true });
        },
    });

    vscode.workspace.onDidChangeConfiguration((e) => {
        if (e.affectsConfiguration('bugbuddy.path')) {
            // prompt user first if they want to restart server
            vscode.window.showInformationMessage('BugBuddy server path has changed. Do you want to restart the server?', 'Yes', 'No')
                .then((value) => {
                    // restart server if server settings change
                    if (value === 'Yes') {
                        disconnectServer();
                        startServer();
                    }
                });
        }
    });

	initializeStatusBar();
	startServer();
}

export async function deactivate() {
	await disposeTerminal();
	await disconnectServer();
}
