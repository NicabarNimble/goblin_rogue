# GitHub Actions Sync Documentation

## Overview

The GitHub Actions sync feature provides automated repository synchronization through GitHub Actions workflows. It enables you to keep repositories in sync by automatically pulling changes from a source repository to a target repository on a scheduled basis or on-demand.

## How It Works

The sync process is implemented as a GitHub Actions workflow that:
1. Checks out the target repository
2. Configures Git settings
3. Pulls changes from the source repository
4. Handles any conflicts according to configured error handling policies
5. Pushes changes to the target repository

## Configuration

### Basic Configuration
```json
{
  "sourceRepo": "owner/repo",
  "targetRepo": "fork/repo",
  "schedule": "0 */6 * * *",
  "branchMappings": {
    "main": "main",
    "develop": "dev"
  },
  "errorHandling": {
    "notifyOnError": true,
    "retryAttempts": 3,
    "retryDelay": "5m"
  }
}
```

### Configuration Options

- `sourceRepo`: The repository to sync from (format: owner/repo)
- `targetRepo`: The repository to sync to (format: owner/repo)
- `schedule`: Cron expression for automated sync (default: every 6 hours)
- `branchMappings`: Map of source branches to target branches
- `errorHandling`: Configuration for error handling behavior
  - `notifyOnError`: Whether to send notifications on failure
  - `retryAttempts`: Number of retry attempts on failure
  - `retryDelay`: Delay between retry attempts

## Command Usage

### Initialize Sync

Sets up the GitHub Actions workflow in your repository:

```bash
go-gitsync init --source user/repo --target fork/repo
```

Options:
- `--source`: Source repository (required)
- `--target`: Target repository (required)
- `--schedule`: Custom sync schedule (optional)
- `--branch-map`: Branch mappings (optional)

### Run Sync

Triggers a sync workflow manually:

```bash
go-gitsync run --repo user/repo
```

Options:
- `--repo`: Repository to sync (required)
- `--branch`: Specific branch to sync (optional)

### Check Status

Checks the status of sync workflows:

```bash
go-gitsync status --repo user/repo
```

Options:
- `--repo`: Repository to check (required)
- `--run-id`: Specific run ID to check (optional)
- `--watch`: Watch status updates in real-time (optional)

### View Logs

Retrieves logs from sync workflow runs:

```bash
go-gitsync logs --repo user/repo --run-id 12345
```

Options:
- `--repo`: Repository to get logs from (required)
- `--run-id`: Workflow run ID (required)
- `--follow`: Stream logs in real-time (optional)

### Configure Settings

Updates sync configuration:

```bash
go-gitsync configure --repo user/repo --schedule "0 0 * * *"
```

Options:
- `--repo`: Repository to configure (required)
- `--schedule`: New sync schedule (optional)
- `--branch-map`: Update branch mappings (optional)
- `--error-notify`: Toggle error notifications (optional)

## Error Handling

The sync process includes robust error handling:

1. **Retry Logic**: Failed operations are retried according to configuration
2. **Conflict Resolution**: Automatic handling of merge conflicts based on strategy
3. **Notifications**: Optional notifications on sync failures
4. **Logging**: Detailed logs for troubleshooting

## Troubleshooting

Common issues and solutions:

1. **Authentication Errors**
   - Ensure GitHub token has required permissions
   - Verify token hasn't expired
   - Check token scope includes workflow and repo access

2. **Workflow Failures**
   - Check workflow logs for detailed error messages
   - Verify branch mappings are correct
   - Ensure source repository is accessible

3. **Configuration Issues**
   - Validate JSON syntax in config file
   - Check repository names are in correct format
   - Verify cron schedule expression is valid

4. **Network Problems**
   - Check GitHub API status
   - Verify network connectivity
   - Ensure firewall allows GitHub API access

## Best Practices

1. **Schedule Selection**
   - Choose sync schedules that align with repository activity
   - Avoid scheduling at peak traffic times
   - Consider time zones when setting schedules

2. **Branch Management**
   - Map branches explicitly for clarity
   - Keep branch names consistent
   - Document branch mapping strategy

3. **Error Handling**
   - Configure appropriate retry attempts
   - Set up notifications for critical syncs
   - Monitor sync status regularly

4. **Security**
   - Use minimal required permissions
   - Rotate GitHub tokens periodically
   - Audit sync configurations regularly
