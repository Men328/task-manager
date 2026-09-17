import i18n from '../../i18n';
import { useAppDispatch, useAppSelector } from '../hooks';
import {
  fetchStatusesPage,
  seedStatusesPage,
  selectPageError,
  selectPageLoading,
  selectPageStatusById,
  selectPageStatuses,
  selectPageTasks,
  selectPageTransitions,
} from './statusesSlice';

export function useStatuses() {
  const dispatch = useAppDispatch();
  const statuses = useAppSelector(selectPageStatuses);
  const transitions = useAppSelector(selectPageTransitions);
  const tasks = useAppSelector(selectPageTasks);
  const statusById = useAppSelector(selectPageStatusById);
  const loading = useAppSelector(selectPageLoading);
  const error = useAppSelector(selectPageError);
  const profileId = useAppSelector((state) => state.session.profileId);

  return {
    statuses,
    transitions,
    tasks,
    statusById,
    loading,
    error,
    refresh: () => {
      if (profileId) {
        void dispatch(fetchStatusesPage(profileId));
      }
    },
    seedDefaultStatuses: async () => {
      if (!profileId) {
        throw new Error(i18n.t('errors.noProfile'));
      }
      await dispatch(seedStatusesPage(profileId)).unwrap();
    },
  };
}
