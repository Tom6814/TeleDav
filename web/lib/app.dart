import 'package:flutter/material.dart';

import 'api_client.dart';
import 'models.dart';
import 'screens/files_screen.dart';
import 'screens/jobs_screen.dart';
import 'screens/login_screen.dart';
import 'screens/settings_screen.dart';

class NetdiskApp extends StatefulWidget {
  const NetdiskApp({super.key});

  @override
  State<NetdiskApp> createState() => _NetdiskAppState();
}

class _NetdiskAppState extends State<NetdiskApp> {
  final ApiClient _api = ApiClient();
  final TextEditingController _passwordController = TextEditingController();

  bool _authenticated = false;
  String? _errorMessage;
  List<String> _directories = const [];
  List<String> _files = const [];
  List<PendingJob> _jobs = const [];
  StorageConfig _config = const StorageConfig();

  @override
  void dispose() {
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _login() async {
    try {
      await _api.login(_passwordController.text);
      final tree = await _api.fetchTree();
      final listing = tree['listing'] as Map<String, dynamic>? ?? {};
      final directories = (listing['directories'] as List<dynamic>? ?? [])
          .map((entry) => (entry as Map<String, dynamic>)['name'] as String? ?? '')
          .toList();
      final files = (listing['files'] as List<dynamic>? ?? [])
          .map((entry) => (entry as Map<String, dynamic>)['name'] as String? ?? '')
          .toList();
      final config = await _api.fetchStorageConfig();
      final jobs = await _api.fetchJobs();

      setState(() {
        _authenticated = true;
        _directories = directories;
        _files = files;
        _config = config;
        _jobs = jobs;
        _errorMessage = null;
      });
    } catch (error) {
      setState(() {
        _errorMessage = error.toString();
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Telegram WebDAV Netdisk',
      theme: ThemeData(useMaterial3: true, colorSchemeSeed: Colors.blue),
      home: _authenticated
          ? DefaultTabController(
              length: 3,
              child: Scaffold(
                appBar: AppBar(
                  title: const Text('Telegram WebDAV Netdisk'),
                  bottom: const TabBar(
                    tabs: [
                      Tab(text: 'Files'),
                      Tab(text: 'Settings'),
                      Tab(text: 'Jobs'),
                    ],
                  ),
                ),
                body: TabBarView(
                  children: [
                    FilesScreen(directories: _directories, files: _files),
                    SettingsScreen(config: _config),
                    JobsScreen(jobs: _jobs),
                  ],
                ),
              ),
            )
          : LoginScreen(
              controller: _passwordController,
              onSubmit: _login,
              errorMessage: _errorMessage,
            ),
    );
  }
}
