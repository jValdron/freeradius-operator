import Logger from 'pino';

import config from '../Config';

export const logger = Logger({
  name: 'freeradius-operator',
  level: config.LogLevel
});