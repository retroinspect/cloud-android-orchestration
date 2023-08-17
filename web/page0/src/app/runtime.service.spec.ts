import { HttpClientTestingModule } from '@angular/common/http/testing';
import { TestBed } from '@angular/core/testing';

import { RuntimeService } from './runtime.service';

describe('RuntimeService', () => {
  let service: RuntimeService;

  beforeEach(() => {
    TestBed.configureTestingModule({
      imports: [HttpClientTestingModule],
      providers: [],
    });

    service = TestBed.inject(RuntimeService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
