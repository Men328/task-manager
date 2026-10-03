import { configureStore } from '@reduxjs/toolkit';

import { boardReducer } from './board/boardSlice';
import { calendarReducer } from './calendar/calendarSlice';
import { sessionReducer } from './session/sessionSlice';
import { statusesPageReducer } from './statuses/statusesSlice';

export const store = configureStore({
  reducer: {
    session: sessionReducer,
    board: boardReducer,
    calendar: calendarReducer,
    statusesPage: statusesPageReducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
export type AppStore = typeof store;
