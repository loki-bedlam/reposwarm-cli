import { Command } from 'commander'
import chalk from 'chalk'
import { getGlobalOpts, output, error, success } from '../lib/output.js'
import { listTrackedRepos, addTrackedRepo, removeTrackedRepo, updateTrackedRepo } from '../lib/aws.js'
import type { RepoInfo } from '../lib/aws.js'

export const reposCommand = new Command('repos')
  .description('Manage tracked repositories')

reposCommand
  .command('list')
  .description('List all tracked repositories')
  .option('--source <source>', 'Filter by source (CodeCommit, GitHub)')
  .option('--enabled', 'Show only enabled repos')
  .option('--disabled', 'Show only disabled repos')
  .option('--filter <text>', 'Filter by name (case-insensitive contains)')
  .action(async (opts, cmd) => {
    const global = getGlobalOpts(cmd)
    try {
      let repos = await listTrackedRepos(global)

      if (opts.source) repos = repos.filter(r => r.source.toLowerCase() === opts.source.toLowerCase())
      if (opts.enabled) repos = repos.filter(r => r.enabled)
      if (opts.disabled) repos = repos.filter(r => !r.enabled)
      if (opts.filter) {
        const f = opts.filter.toLowerCase()
        repos = repos.filter(r => r.name.toLowerCase().includes(f))
      }

      output(cmd, { count: repos.length, repos }, (data) => {
        if (data.count === 0) {
          console.log('No repositories found. Run `reposwarm discover` to auto-discover.')
          return
        }
        console.log(`\n${chalk.bold(data.count)} repositories:\n`)
        for (const repo of data.repos) {
          const status = repo.enabled ? chalk.green('●') : chalk.dim('○')
          const source = chalk.dim(`[${repo.source}]`)
          console.log(`  ${status} ${repo.name} ${source}`)
        }
        console.log()
      })
    } catch (err) {
      error(cmd, 'Failed to list repos', err)
    }
  })

reposCommand
  .command('add <name>')
  .description('Add a repository to tracking')
  .requiredOption('--url <url>', 'Repository URL')
  .option('--source <source>', 'Source type (CodeCommit, GitHub)', 'CodeCommit')
  .action(async (name, opts, cmd) => {
    const global = getGlobalOpts(cmd)
    try {
      const repo: RepoInfo = {
        name,
        url: opts.url,
        source: opts.source as 'CodeCommit' | 'GitHub',
        enabled: true,
        status: 'active'
      }
      await addTrackedRepo(global, repo)
      output(cmd, { success: true, repo }, () => {
        success(`Added ${name}`)
      })
    } catch (err) {
      error(cmd, `Failed to add ${name}`, err)
    }
  })

reposCommand
  .command('remove <name>')
  .description('Remove a repository from tracking')
  .action(async (name, _opts, cmd) => {
    const global = getGlobalOpts(cmd)
    try {
      await removeTrackedRepo(global, name)
      output(cmd, { success: true, name }, () => {
        success(`Removed ${name}`)
      })
    } catch (err) {
      error(cmd, `Failed to remove ${name}`, err)
    }
  })

reposCommand
  .command('enable <name>')
  .description('Enable a repository for investigation')
  .action(async (name, _opts, cmd) => {
    const global = getGlobalOpts(cmd)
    try {
      await updateTrackedRepo(global, name, { enabled: true })
      output(cmd, { success: true, name, enabled: true }, () => {
        success(`Enabled ${name}`)
      })
    } catch (err) {
      error(cmd, `Failed to enable ${name}`, err)
    }
  })

reposCommand
  .command('disable <name>')
  .description('Disable a repository from investigation')
  .action(async (name, _opts, cmd) => {
    const global = getGlobalOpts(cmd)
    try {
      await updateTrackedRepo(global, name, { enabled: false })
      output(cmd, { success: true, name, enabled: false }, () => {
        success(`Disabled ${name}`)
      })
    } catch (err) {
      error(cmd, `Failed to disable ${name}`, err)
    }
  })
