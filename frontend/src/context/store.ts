import { configureStore } from '@reduxjs/toolkit';

import { backlogReducer } from './backlog/backlogSlice';
import { boardReducer } from './board/boardSlice';
import { calendarReducer } from './calendar/calendarSlice';
import { eventReducer } from './event/eventSlice';
import { sessionReducer } from './session/sessionSlice';
import { statusesPageReducer } from './statuses/statusesSlice';

export const store = configureStore({
  reducer: {
    session: sessionReducer,
    board: boardReducer,
    calendar: calendarReducer,
    event: eventReducer,
    backlog: backlogReducer,
    statusesPage: statusesPageReducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
export type AppStore = typeof store;
