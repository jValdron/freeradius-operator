require('dotenv').config();

import FreeRadiusOperator from './FreeRadiusOperator'

import { logger } from './utils/Logger';

(async () => {
  const operator = new FreeRadiusOperator();
  logger.info({ operator }, 'Starting operator')
  operator.start();

  const exit = (reason: string) => {
    logger.info({
      reason: reason
    }, 'Stopping operator');

    operator.stop();

    process.exit(0);
  };
  
  process.on('unhandledRejection', (reason, promise) => {
    logger.error({
      reason: reason,
      promise: promise
    }, 'Unhandled rejection within promise');
  });
  
  process.on('SIGTERM', () => exit('SIGTERM'))
         .on('SIGINT', () => exit('SIGINT'));
})();