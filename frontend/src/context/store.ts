import { configureStore } from '@reduxjs/toolkit';

import { backlogReducer } from './backlog/backlogSlice';
import { boardReducer } from './board/boardSlice';
import { calendarReducer } from './calendar/calendarSlice';
import { eventReducer } from './event/eventSlice';
import { noticeReducer } from './notification/noticeSlice';
import { projectReducer } from './project/projectSlice';
import { reportReducer } from './report/reportSlice';
import { sessionReducer } from './session/sessionSlice';
import { statusesPageReducer } from './statuses/statusesSlice';

export const store = configureStore({
  reducer: {
    session: sessionReducer,
    board: boardReducer,
    calendar: calendarReducer,
    event: eventReducer,
    backlog: backlogReducer,
    project: projectReducer,
    report: reportReducer,
    statusesPage: statusesPageReducer,
    notices: noticeReducer,
  },
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;
export type AppStore = typeof store;
