import { ChildProcessWithoutNullStreams, spawn } from "child_process";
import { commands, window } from "vscode";
import { LanguageClient, LanguageClientOptions, ServerOptions } from "vscode-languageclient/node";
import { ConnectionStatus, extensionId, getWorkspaceConfig, outputChannel, setConnectionStatus } from "./utils";

let serverProcess: ChildProcessWithoutNullStreams;
let client: LanguageClient;

export class ClientError extends Error {
    constructor(message: string) {
        super(message);
        this.name = 'ClientError';
    }
}

export function getClient(): LanguageClient {
    if (!client) {
        throw new ClientError('BugBuddy LSP client not initialized.');
    }
    return client;
}

export function launchServer(execPath: string) {
	const proc = spawn(execPath, ["lsp"], {shell: true});
    proc.stderr.on('data', (raw: string | Buffer) => {
		if (raw instanceof Buffer) {
			outputChannel.appendLine(raw.toString('utf-8'));
		} else {
			outputChannel.appendLine(raw);
		}
	});
    return proc;
}

export function initializeServer() {
    if (client && client.needsStop()) {
        // do not reinitialize the client if it's already running
        window.showErrorMessage('BugBuddy LSP client is already running.');
        return client;
    }

    const customPath = getWorkspaceConfig().get<string>('path', 'bugbuddy');
	console.log('Launching BugBuddy from', customPath);

	serverProcess = launchServer(customPath);

	const serverOpts: ServerOptions = () => Promise.resolve(serverProcess);
	const clientOpts: LanguageClientOptions = {
		documentSelector: [{ scheme: 'file' }]
	};

	client = new LanguageClient('BugBuddy LSP', serverOpts, clientOpts);
    return client;
}

export async function startServer() {
    try {
        setConnectionStatus(ConnectionStatus.connecting);
        const client = initializeServer();
        await client.start();

        // get participant id
        try {
            // eslint-disable-next-line @typescript-eslint/naming-convention
            const got = await client.sendRequest<{ participant_id: string }>('$/participantId');
            setConnectionStatus(ConnectionStatus.connected, { participantId: got.participant_id });
        } catch (e) {
            setConnectionStatus(ConnectionStatus.connected, { participantId: 'unknown' });
        }
    } catch (e) {
        setConnectionStatus(ConnectionStatus.failed);
        window.showErrorMessage(`Failed to connect: ${e}`);
    }
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
    try {
        // set status bar to disconnected
        setConnectionStatus(ConnectionStatus.disconnected);

        // remove active bugbuddy markdown preview
        disposeErrorSection();

        const client = getClient();
        await client.stop();
        await client.dispose();
    } catch (e) {
        if (e instanceof ClientError) {
            outputChannel.appendLine(`[BugBuddy - ERROR] ${e}`);
        } else {
            throw e;
        }
    }
}

// Language server stats menu
interface ServerStats {
    daemon: {
        // eslint-disable-next-line @typescript-eslint/naming-convention
        is_connected: boolean
        port: number
    }

    // eslint-disable-next-line @typescript-eslint/naming-convention
    participant_id: string
    version: string
}

export async function showServerMenu() {
    let stats: ServerStats | null = null;

    try {
        const client = getClient();
        if (client.needsStart()) {
            throw new ClientError('BugBuddy LSP client not initialized.');
        }

        stats = await client.sendRequest<ServerStats>('$/status');
    } catch (e) {
        if (!(e instanceof ClientError)) {
            throw e;
        }
    } finally {
        const picked = await window.showQuickPick([
            `BugBuddy Version: ${stats ? stats.version : 'unknown'}`,
            `Participant ID: ${stats ? stats.participant_id : 'unknown'}`,
            `Daemon: ${stats && stats.daemon.is_connected ? `Connected at port ${stats.daemon.port}` : 'Disconnected'}`,
            ...(stats ? [
                `Generate new participant ID`,
                'Disconnect server'
            ] : [
                'Connect server'
            ])
        ], {
            canPickMany: false
        });

        switch (picked) {
            case 'Connect server':
                await startServer();
                break;
            case 'Disconnect server':
                await disconnectServer();
                break;
        }
    }
}

export async function generateParticipantId() {
    const resp = await window.showInformationMessage('Are you sure you want to generate a new participant ID?', 'Yes', 'No');

    // eslint-disable-next-line @typescript-eslint/naming-convention
    const got = await getClient().sendRequest<{ participant_id: string }>('$/participantId/generate', { confirm: resp === 'Yes' });
    setConnectionStatus(ConnectionStatus.connected, { participantId: got.participant_id });
    window.showInformationMessage('A new participant ID has been generated.');
}
