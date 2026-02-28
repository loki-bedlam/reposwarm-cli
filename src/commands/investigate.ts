import { Command } from 'commander'
import chalk from 'chalk'
import { getGlobalOpts, output, error, success, info } from '../lib/output.js'
import { listTrackedRepos, listCodeCommitRepos, addTrackedRepo } from '../lib/aws.js'

interface TemporalStartResult {
  workflowId: string
  runId?: string
}

async function startTemporalWorkflow(
  temporalUrl: string,
  namespace: string,
  taskQueue: string,
  workflowId: string,
  workflowType: string,
  args: unknown[]
): Promise<TemporalStartResult> {
  const response = await fetch(`${temporalUrl}/api/v1/namespaces/${namespace}/workflows/${encodeURIComponent(workflowId)}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      workflowType: { name: workflowType },
      taskQueue: { name: taskQueue },
      input: { payloads: args.map(a => ({ data: Buffer.from(JSON.stringify(a)).toString('base64'), metadata: { encoding: 'anNvbi9wbGFpbg==' } })) }
    })
  })
  if (!response.ok) {
    const text = await response.text()
    throw new Error(`Temporal API error (${response.status}): ${text}`)
  }
  const data = await response.json() as { runId?: string }
  return { workflowId, runId: data.runId }
}

export const investigateCommand = new Command('investigate')
  .description('Trigger architecture investigation')
  .argument('[repo]', 'Repository name (omit for all)')
  .option('--all', 'Investigate all enabled repos')
  .option('--discover', 'Auto-discover repos before investigating')
  .option('--model <model>', 'Model ID', 'us.anthropic.claude-sonnet-4-6')
  .option('--chunk-size <n>', 'Chunk size', '10')
  .option('--parallel <n>', 'Parallel limit (daily only)', '3')
  .option('--temporal-url <url>', 'Temporal HTTP API URL', 'http://temporal-alb-internal:8233')
  .option('--namespace <ns>', 'Temporal namespace', 'default')
  .option('--task-queue <queue>', 'Temporal task queue', 'investigate-task-queue')
  .action(async (repo, opts, cmd) => {
    const global = getGlobalOpts(cmd)
    try {
      // Auto-discover if requested
      if (opts.discover) {
        info('Auto-discovering repos from CodeCommit...')
        const discovered = await listCodeCommitRepos(global)
        const existing = await listTrackedRepos(global)
        const existingNames = new Set(existing.map(r => r.name))
        let added = 0
        for (const r of discovered) {
          if (!existingNames.has(r.name)) {
            await addTrackedRepo(global, r)
            added++
          }
        }
        if (added > 0) info(`Discovered ${discovered.length} repos, added ${added} new`)
      }

      if (repo) {
        // Single repo investigation
        const workflowId = `investigate-single-${repo}-${Date.now()}`
        const result = await startTemporalWorkflow(
          opts.temporalUrl, opts.namespace, opts.taskQueue,
          workflowId, 'InvestigateSingleRepoWorkflow',
          [{ repoName: repo, model: opts.model, chunkSize: parseInt(opts.chunkSize) }]
        )
        output(cmd, { success: true, ...result, repo }, () => {
          success(`Investigation started for ${chalk.bold(repo)}`)
          console.log(`  Workflow: ${result.workflowId}`)
        })
      } else if (opts.all) {
        // Daily/all repos investigation
        const repos = await listTrackedRepos(global)
        const enabled = repos.filter(r => r.enabled)

        if (enabled.length === 0) {
          error(cmd, 'No enabled repos. Run `reposwarm discover` or `reposwarm investigate --all --discover`')
          return
        }

        const repoNames = enabled.map(r => r.name)
        const workflowId = `investigate-daily-${Date.now()}`
        const result = await startTemporalWorkflow(
          opts.temporalUrl, opts.namespace, opts.taskQueue,
          workflowId, 'InvestigateReposWorkflow',
          [{ repos: repoNames, model: opts.model, chunkSize: parseInt(opts.chunkSize), parallelLimit: parseInt(opts.parallel) }]
        )
        output(cmd, { success: true, ...result, repoCount: repoNames.length, repos: repoNames }, () => {
          success(`Daily investigation started for ${chalk.bold(String(repoNames.length))} repos`)
          console.log(`  Workflow: ${result.workflowId}`)
        })
      } else {
        error(cmd, 'Specify a repo name or use --all. Example: reposwarm investigate my-repo')
      }
    } catch (err) {
      error(cmd, 'Investigation failed', err)
    }
  })
