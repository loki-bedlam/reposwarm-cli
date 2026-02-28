#!/usr/bin/env node
import { Command } from 'commander'
import { discoverCommand } from './commands/discover.js'
import { reposCommand } from './commands/repos.js'
import { investigateCommand } from './commands/investigate.js'
import { workflowsCommand } from './commands/workflows.js'
import { configCommand } from './commands/config.js'

const program = new Command()

program
  .name('reposwarm')
  .description('CLI for RepoSwarm — AI-powered multi-repo architecture discovery')
  .version('0.1.0')
  .option('--json', 'Output as JSON (agent-friendly)')
  .option('--region <region>', 'AWS region', 'us-east-1')
  .option('--profile <profile>', 'AWS profile')
  .option('--table <table>', 'DynamoDB table name', 'reposwarm-cache')

program.addCommand(discoverCommand)
program.addCommand(reposCommand)
program.addCommand(investigateCommand)
program.addCommand(workflowsCommand)
program.addCommand(configCommand)

program.parse()
