import { Command } from 'commander'
import chalk from 'chalk'
import { getGlobalOpts, output } from '../lib/output.js'

export const configCommand = new Command('config')
  .description('Show current configuration')
  .action(async (_opts, cmd) => {
    const global = getGlobalOpts(cmd)
    const config = {
      region: global.region,
      profile: global.profile || '(default)',
      table: global.table,
      temporalUrl: 'http://temporal-alb-internal:8233',
      namespace: 'default',
      taskQueue: 'investigate-task-queue'
    }

    output(cmd, config, (data) => {
      console.log(`\n${chalk.bold('RepoSwarm CLI Configuration')}\n`)
      for (const [key, value] of Object.entries(data)) {
        console.log(`  ${chalk.dim(key.padEnd(16))} ${value}`)
      }
      console.log()
    })
  })
