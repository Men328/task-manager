import { configureStore } from '@reduxjs/toolkit';

import { boardReducer } from './board/boardSlice';
import { sessionReducer } from './session/sessionSlice';
import { statusesPageReducer } from './statuses/statusesSlice';
import { workspaceReducer } from './workspace/workspaceSlice';

export const store = configureStore({
  reducer: {
    session: sessionReducer,
    workspace: workspaceReducer,
    board: boardReducer,
    statusesPage: statusesPageReducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
export type AppStore = typeof store;
