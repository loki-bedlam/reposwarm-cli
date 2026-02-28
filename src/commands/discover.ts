import { Command } from 'commander'
import chalk from 'chalk'
import { getGlobalOpts, output, error, success, info } from '../lib/output.js'
import { listCodeCommitRepos, listTrackedRepos, addTrackedRepo } from '../lib/aws.js'

export const discoverCommand = new Command('discover')
  .description('Auto-discover repositories from CodeCommit (and GitHub in future)')
  .option('--source <source>', 'Source to discover from (codecommit)', 'codecommit')
  .option('--dry-run', 'Show what would be added without persisting')
  .option('--force', 'Re-add repos even if already tracked')
  .action(async (opts, cmd) => {
    const global = getGlobalOpts(cmd)
    try {
      info(`Discovering repos from ${opts.source}...`)

      const discovered = await listCodeCommitRepos(global)
      const existing = await listTrackedRepos(global)
      const existingNames = new Set(existing.map(r => r.name))

      let added = 0
      let skipped = 0

      for (const repo of discovered) {
        if (!opts.force && existingNames.has(repo.name)) {
          skipped++
          continue
        }
        if (!opts.dryRun) {
          await addTrackedRepo(global, repo)
        }
        added++
      }

      const result = {
        discovered: discovered.length,
        added,
        skipped,
        dryRun: opts.dryRun || false,
        repositories: discovered.map(r => ({
          name: r.name,
          url: r.url,
          source: r.source,
          isNew: !existingNames.has(r.name)
        }))
      }

      output(cmd, result, (data) => {
        console.log(`\nFound ${chalk.bold(data.discovered)} repositories:`)
        for (const repo of data.repositories) {
          const tag = repo.isNew ? chalk.green(' NEW') : chalk.dim(' exists')
          console.log(`  ${repo.name}${tag}`)
        }
        console.log()
        if (data.dryRun) {
          info(`Dry run: would add ${data.added} repos (${data.skipped} already tracked)`)
        } else {
          success(`Added ${data.added} new repos (${data.skipped} already tracked)`)
        }
      })
    } catch (err) {
      error(cmd, 'Discovery failed', err)
    }
  })
