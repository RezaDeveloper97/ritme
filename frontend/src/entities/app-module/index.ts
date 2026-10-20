// Public API of the `app-module` entity — the admin kill-switches for whole app modules. Import only from here.

export {
  ALL_MODULES_ON,
  APP_MODULES,
  modeAvailable,
  moduleOfRoute,
  type AppModule,
  type AppModules,
} from './model/modules';
export { appModuleKeys, fetchAppModules, useAppModules, useAppModulesQuery, useModuleEnabled } from './api/queries';
