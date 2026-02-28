import { Command } from 'commander'
import chalk from 'chalk'
import { getGlobalOpts, output, error } from '../lib/output.js'

interface WorkflowExecution {
  workflowId: string
  runId: string
  status: string
  type: string
  startTime: string
  closeTime?: string
}

async function fetchWorkflows(temporalUrl: string, namespace: string, query?: string): Promise<WorkflowExecution[]> {
  const params = new URLSearchParams()
  if (query) params.set('query', query)

  const response = await fetch(
    `${temporalUrl}/api/v1/namespaces/${namespace}/workflows?${params}`,
    { headers: { 'Content-Type': 'application/json' } }
  )
  if (!response.ok) throw new Error(`Temporal API error: ${response.status}`)

  const data = await response.json() as { executions?: Array<{
    execution: { workflowId: string; runId: string }
    status: string
    type: { name: string }
    startTime: string
    closeTime?: string
  }> }
  return (data.executions || []).map(e => ({
    workflowId: e.execution.workflowId,
    runId: e.execution.runId,
    status: e.status,
    type: e.type.name,
    startTime: e.startTime,
    closeTime: e.closeTime
  }))
}

export const workflowsCommand = new Command('workflows')
  .description('Manage investigation workflows')

workflowsCommand
  .command('list')
  .description('List recent workflows')
  .option('--status <status>', 'Filter by status (Running, Completed, Failed)')
  .option('--limit <n>', 'Max results', '20')
  .option('--temporal-url <url>', 'Temporal HTTP API URL', 'http://temporal-alb-internal:8233')
  .option('--namespace <ns>', 'Temporal namespace', 'default')
  .action(async (opts, cmd) => {
    const global = getGlobalOpts(cmd)
    try {
      let query = ''
      if (opts.status) {
        const statusMap: Record<string, string> = {
          running: 'Running', completed: 'Completed', failed: 'Failed',
          terminated: 'Terminated', timedout: 'TimedOut'
        }
        const s = statusMap[opts.status.toLowerCase()] || opts.status
        query = `ExecutionStatus = "${s}"`
      }

      const workflows = await fetchWorkflows(opts.temporalUrl, opts.namespace, query)

      output(cmd, { count: workflows.length, workflows }, (data) => {
        if (data.count === 0) {
          console.log('No workflows found.')
          return
        }
        console.log(`\n${chalk.bold(data.count)} workflows:\n`)
        for (const wf of data.workflows) {
          const statusColor = wf.status === 'Running' ? chalk.blue
            : wf.status === 'Completed' ? chalk.green
            : wf.status === 'Failed' ? chalk.red
            : chalk.dim
          console.log(`  ${statusColor(wf.status.padEnd(12))} ${wf.workflowId}`)
          console.log(`  ${chalk.dim(`Started: ${wf.startTime}`)}`)
        }
        console.log()
      })
    } catch (err) {
      error(cmd, 'Failed to list workflows', err)
    }
  })

workflowsCommand
  .command('status <workflowId>')
  .description('Get workflow status')
  .option('--temporal-url <url>', 'Temporal HTTP API URL', 'http://temporal-alb-internal:8233')
  .option('--namespace <ns>', 'Temporal namespace', 'default')
  .action(async (workflowId, opts, cmd) => {
    const global = getGlobalOpts(cmd)
    try {
      const response = await fetch(
        `${opts.temporalUrl}/api/v1/namespaces/${opts.namespace}/workflows/${encodeURIComponent(workflowId)}`,
        { headers: { 'Content-Type': 'application/json' } }
      )
      if (!response.ok) throw new Error(`Temporal API error: ${response.status}`)
      const data = await response.json()

      output(cmd, data, (d) => {
        console.log(`\nWorkflow: ${chalk.bold(workflowId)}`)
        console.log(JSON.stringify(d, null, 2))
      })
    } catch (err) {
      error(cmd, `Failed to get workflow ${workflowId}`, err)
    }
  })

workflowsCommand
  .command('terminate <workflowId>')
  .description('Terminate a running workflow')
  .option('--reason <reason>', 'Termination reason', 'Terminated via CLI')
  .option('--temporal-url <url>', 'Temporal HTTP API URL', 'http://temporal-alb-internal:8233')
  .option('--namespace <ns>', 'Temporal namespace', 'default')
  .action(async (workflowId, opts, cmd) => {
    const global = getGlobalOpts(cmd)
    try {
      const response = await fetch(
        `${opts.temporalUrl}/api/v1/namespaces/${opts.namespace}/workflows/${encodeURIComponent(workflowId)}/terminate`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ reason: opts.reason })
        }
      )
      if (!response.ok) throw new Error(`Temporal API error: ${response.status}`)

      output(cmd, { success: true, workflowId, reason: opts.reason }, () => {
        console.log(chalk.yellow(`Terminated workflow: ${workflowId}`))
      })
    } catch (err) {
      error(cmd, `Failed to terminate ${workflowId}`, err)
    }
  })
