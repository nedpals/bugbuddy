import { join as joinWin32 } from "path/win32";
import { join as joinPosix } from "path/posix";
import { homedir } from "os";
import { ChildProcessWithoutNullStreams, spawn } from "child_process";
import { commands, window } from "vscode";
import { LanguageClient, LanguageClientOptions, ServerOptions } from "vscode-languageclient/node";
import { ConnectionStatus, extensionId, getWorkspaceConfig, openExplorerIn, outputChannel, setConnectionStatus } from "./utils";

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
    if (!execPath || execPath.length == 0) {
        return;
    }

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

    const customPath = getWorkspaceConfig().get<string>('path',
        process.platform === 'win32' ?
            joinWin32('C:', 'bugbuddy', 'bugbuddy_windows_amd64.exe') :
            joinPosix(homedir(), 'bugbuddy', process.platform === 'darwin' ? 'bugbuddy_macos_universal' : 'bugbuddy_linux_amd64'));
	console.log('Launching BugBuddy from', customPath);

	const newServerProcess = launchServer(customPath);
    if (!newServerProcess) {
        // Do not continue if server process is not created
        return;
    }

    serverProcess = newServerProcess;

	const serverOpts: ServerOptions = () => Promise.resolve(serverProcess);
	const clientOpts: LanguageClientOptions = {
		documentSelector: [{ scheme: 'file' }],
        initializationOptions: {
            // eslint-disable-next-line @typescript-eslint/naming-convention
            data_dir_path: getWorkspaceConfig().get('dataDirPath'),
            // eslint-disable-next-line @typescript-eslint/naming-convention
            daemon_port: getWorkspaceConfig().get('daemonPort', 3434),
        }
	};

	client = new LanguageClient('BugBuddy LSP', serverOpts, clientOpts);
    return client;
}

export async function startServer() {
    try {
        setConnectionStatus(ConnectionStatus.connecting);
        const client = initializeServer();
        if (!client) {
            setConnectionStatus(ConnectionStatus.disconnected);
            return;
        }

        await client.start();

        // get participant id
        const participantId = await getParticipantId();
        setConnectionStatus(ConnectionStatus.connected, { participantId });
    } catch (e) {
        setConnectionStatus(ConnectionStatus.failed);
        window.showErrorMessage(`Failed to connect: ${e}`);
    }
}

export async function setDataDirPath(newPath: string) {
    if (!newPath) {
        // do not continue if path is empty
        return;
    }

    if (!isServerConnected()) {
        window.showErrorMessage('BugBuddy server is not connected. Please connect to the server first.');
        return;
    }

    // change data dir path if server is connected
    const client = getClient();
    await client.sendRequest('$/setDataDir', {
        // eslint-disable-next-line @typescript-eslint/naming-convention
        new_path: newPath
    });
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
        const client = getClient();

        // if client is not initialized, do nothing
        if (!isServerConnected()) {
            return;
        }

        // set status bar to disconnected
        setConnectionStatus(ConnectionStatus.disconnected);

        // remove active bugbuddy markdown preview
        disposeErrorSection();

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

export function isServerConnected() {
    const client = getClient();
    if (!client) {
        return false;
    }
    return client.needsStop();
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
    const client = getClient();

    try {
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
                'Open BugBuddy data directory',
                'Generate new participant ID',
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
            case 'Generate new participant ID':
                await generateParticipantId();
                break;
            case 'Open BugBuddy data directory':
                // eslint-disable-next-line @typescript-eslint/naming-convention
                const resp = await client.sendRequest<{ data_dir: string }>('$/fetchDataDir');
                openExplorerIn(resp.data_dir, (err) => {
                    if (err) {
                        window.showErrorMessage(`Failed to open directory: ${err}`);
                    }
                });
                break;
        }
    }
}

export async function getParticipantId() {
    try {
        const got = await client.sendRequest<{ participant_id: string }>('$/participantId');
        return got.participant_id;
    } catch (e) {
        return 'unknown';
    }
}

export async function generateParticipantId() {
    const resp = await window.showInformationMessage('Are you sure you want to generate a new participant ID?', 'Yes', 'No');

    // eslint-disable-next-line @typescript-eslint/naming-convention
    const got = await getClient().sendRequest<{ participant_id: string }>('$/participantId/generate', { confirm: resp === 'Yes' });
    setConnectionStatus(ConnectionStatus.connected, { participantId: got.participant_id });
    window.showInformationMessage('A new participant ID has been generated.');
}
